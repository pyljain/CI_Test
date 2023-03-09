package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"gopkg.in/yaml.v3"
)

func main() {

	flag.Parse()
	dirName := flag.Arg(0)

	if dirName == "" {
		log.Fatal("Please pass in a root directory argument")
		os.Exit(1)
	}

	err := loadAndParse(dirName)
	if err != nil {
		log.Fatalf("%s", err)
	}

	// log.Printf("Argument received %s", dirName)
}

func loadAndParse(dirName string) error {
	// Check if Values.yaml exists

	valuesYAMLLoc := fmt.Sprintf("%s/values.yaml", dirName)

	_, err := os.Stat(valuesYAMLLoc)
	if err != nil {
		return fmt.Errorf("values.yaml not found")
	}

	// If exists, read values.yaml
	valuesInBytes, err := ioutil.ReadFile(valuesYAMLLoc)
	if err != nil {
		return fmt.Errorf("Unable to parse values.yaml")
	}

	var values map[string]interface{}

	err = yaml.Unmarshal(valuesInBytes, &values)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal values.yaml")
	}

	// log.Printf("VALUES UNMARSHALLED %+v", values)

	// Go through all YAMLs and check if replacements are needed

	filesToProcess := []string{}

	filepath.Walk(dirName, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if info.Name() == "values.yaml" {
			return nil
		}

		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			filesToProcess = append(filesToProcess, path)
		}

		return nil
	})

	log.Printf("FILES TO PROCESS %+v", filesToProcess)

	// Prepare to call function
	wg := sync.WaitGroup{}
	output := []string{}
	lock := sync.Mutex{}

	//Loop through files
	for _, f := range filesToProcess {
		wg.Add(1)
		go ApplyTemplate(f, &wg, &lock, values, &output)
	}

	wg.Wait()
	consolidatedOutputString := strings.Join(output, "\n---\n")
	fmt.Println(consolidatedOutputString)

	return nil
}

func ApplyTemplate(path string, wg *sync.WaitGroup, lock *sync.Mutex, values map[string]interface{}, output *[]string) {

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Error processing file %s", err)
		return
	}

	tmpl, err := template.New(path).Parse(string(fileBytes))
	if err != nil {
		log.Printf("Error creating template %s", err)
		return
	}

	var buf bytes.Buffer
	input := TemplateInput{
		Values: values,
	}

	err = tmpl.Execute(&buf, input)
	if err != nil {
		log.Printf("Error executing template %s", err)
		return
	}

	lock.Lock()
	*output = append(*output, buf.String())
	lock.Unlock()
	wg.Done()

}

type TemplateInput struct {
	Values map[string]interface{}
}
