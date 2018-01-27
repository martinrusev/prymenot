package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
	"gopkg.in/yaml.v2"
)

var path = "/home/martin/go/src/github.com/prymenot/prymenot"

type Sources struct {
	List []Source
}

type Source struct {
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func syncSources() (err error) {
	filename, _ := filepath.Abs("sources.yml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal("Can not open sources file at : %s - %s", filename, err)
	}

	var sources Sources

	err = yaml.Unmarshal(yamlFile, &sources.List)
	if err != nil {
		log.Fatal("Can not process YAML in: %s - %s", filename, err)
	}

	os.MkdirAll("sources", os.ModePerm)

	for _, element := range sources.List {
		log.Infof("Processing URL:%s \n", element.Url)

		outputPath := filepath.Join(path, "sources", element.Name)
		downloadFile(outputPath, element.Url)
	}

	return nil

}

func parseLine(line string) (validURL string, err error) {
	cleanLine := strings.TrimSpace(strings.ToLower(line))

	// Split by tab first
	splitByTab := strings.Split(cleanLine, "\t")

	if len(splitByTab) == 1 {
		splitbySpace := strings.Fields(cleanLine)

		for _, el := range splitbySpace {

			cleanElement := strings.TrimSpace(el)

			isElementURL := govalidator.IsURL(cleanElement)
			isElementIP := govalidator.IsIP(cleanElement)
			startsWithHash := strings.HasPrefix(cleanElement, "#")

			if isElementURL == true && isElementIP == false && startsWithHash == false {
				validURL = cleanElement
			}
		}
	}

	log.Infof("Extracted URL:%s from line: %s\n", validURL, line)

	return validURL, nil
}

func parseFile(file string) (result []string, err error) {

	linesInFile := 0
	linesParsed := 0

	fileToParse := filepath.Join(path, "sources", file)
	log.Infof("Parsing: %s\n", fileToParse)

	openFile, err := os.Open(fileToParse)
	if err != nil {
		log.Fatal("Can not open file: %s - %s", openFile, err)
	}
	defer openFile.Close()

	scanner := bufio.NewScanner(openFile)
	for scanner.Scan() {
		linesInFile = linesInFile + 1
		line := scanner.Text()
		parsedLine, _ := parseLine(line)
		if len(parsedLine) > 0 {
			result = append(result, parsedLine)
			linesParsed = linesParsed + 1
		}
	}

	if err := scanner.Err(); err != nil {
		log.Warn("Can not process file: %s - %s", openFile, err)
	}

	log.Infof("Total lines found in %s: %d | Lines Parsed:%d\n", file, linesInFile, linesParsed)

	return result, nil
}

func main() {

}
