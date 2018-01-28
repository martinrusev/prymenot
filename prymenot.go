package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// Source: https://www.dotnetperls.com/duplicates-go
// :param elements := []string{"cat", "dog", "cat", "bird"}
// Usage::
//   >>> unique = removeDuplicatesUnordered(elements)
//
func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
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

// Function for downloading hosts files
// :param sources: a list of structs [{"url": "http://someonewhocares.org/hosts/zero/hosts", 'name': 'someonewhocarse'}]
// :param output_path: an output directory. absolute path
// Usage::
//   >>> syncSources([{"url": 'http://domain.com/ads_hosts', 'name': 'domain_ads'}])
//
func syncSources(sources Sources, output_path string) (err error) {
	filename, _ := filepath.Abs("sources.yml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal("Can not open sources file at : %s - %s", filename, err)
	}

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

// Function for parsing individual /etc/hosts lines
// :param line: /etc/hosts line
// Usage::
//   >>> host = parseLine(line='127.0.0.1    005.free-counter.co.uk')
//
func parseLine(line string) (validURL string, err error) {
	var lineSplice []string

	cleanLine := strings.TrimSpace(strings.ToLower(line))

	// Split by tab first
	splitByTab := strings.Split(cleanLine, "\t")

	if len(splitByTab) == 1 {
		lineSplice = strings.Fields(cleanLine)
	} else {
		lineSplice = splitByTab
	}

	for _, el := range lineSplice {
		cleanHttp := strings.Replace(el, "http://", "", -1)
		cleanElement := strings.TrimSpace(cleanHttp)

		isElementURL := govalidator.IsURL(cleanElement)
		isElementIP := govalidator.IsIP(cleanElement)
		startsWithHash := strings.HasPrefix(cleanLine, "#")

		if isElementURL == true && isElementIP == false && startsWithHash == false {
			validURL = cleanElement
		}
	}

	log.Debug("Extracted URL:%s from line: %s\n", validURL, line)

	return validURL, nil
}

// Function for parsing /etc/hosts files

// :param path: non relative path to the hosts file
// Usage::
//   >>> hosts_list = parseFile('/etc/hosts')
//
func parseFile(pathToFile string) (results []string, err error) {

	linesInFile := 0
	linesParsed := 0

	openFile, err := os.Open(pathToFile)
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
			results = append(results, parsedLine)
			linesParsed = linesParsed + 1
		}
	}

	if err := scanner.Err(); err != nil {
		log.Warn("Can not process file: %s - %s", openFile, err)
	}

	log.Debug("Total lines found in %s: %d | Lines Parsed: %d\n", pathToFile, linesInFile, linesParsed)

	return results, nil
}

// Function for parsing folders with multiple /etc/hosts files
// :param folderPath: absolute path to a directory with multiple hosts files
// Usage::
//   >>> hosts_list = parseFolder('/home/hosts')
//
func parseFolder(folderPath string) (results []string, err error) {
	log.Infof("Parsing directory %s", folderPath)

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		log.Error("Can not process directory: %s - %s", folderPath, err)
	}

	for _, file := range files {
		if !file.IsDir() {
			pathToFile := filepath.Join(folderPath, file.Name())
			hosts, _ := parseFile(pathToFile)
			results = append(results, hosts...)

		}

	}

	uniqueHosts := removeDuplicatesUnordered(results)
	log.Infof("Total Hosts parsed: %d", len(results))
	log.Infof("Unique Hosts: %d", len(uniqueHosts))

	results = uniqueHosts

	return results, nil
}

// Function for removing dead domains from a list
// :param domains: a list of domains
// Usage::
//   >>> workingDomains = cleanupDeadDomains(domains=['005.free-counter.co.uk', 'warning-0auto7.stream'])
//
func cleanupDeadDomains(domains []string) (result []string, err error) {

	return result, nil
}

// An utility function that exports a domain list to different formats.
//    :param domains: list with hosts(domains)
// 	  :param format: export format. possible options: unbound, json, yaml, hosts
//    :param path: absolute path to the desired location for the generated file
//    :param ip_address: IP Adress to be used in the config, defaults to 0.0.0.0
//    Usage::
//      >>> exportToFile(['advertising.microsoft.com', 'ad.doubleclick.net'], 'yaml', '/home/user/hosts')
//
func exportToFile(domains []string, format string, path string, ip_address string) (err error) {
	if len(ip_address) == 0 {
		ip_address = "0.0.0.0"
	}
	return nil
}

func main() {

	start := time.Now()

	absolutePathToFolder := filepath.Join(path, "sources")
	hosts, _ := parseFolder(absolutePathToFolder)
	log.Debug(hosts)
	elapsed := time.Since(start)
	log.Printf("Parsing hosts files took %s", elapsed)
}
