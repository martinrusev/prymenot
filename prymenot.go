package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var path = "/home/martin/prymenot"

// Sources  - XXX
type Sources struct {
	List []Source
}

// Source  - XXX
type Source struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
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
func syncSources(sources Sources, outputPath string) (err error) {
	filename, _ := filepath.Abs("sources.yml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.WithFields(log.Fields{
			"file":  filename,
			"Error": err,
		}).Fatal("Can not process file")
	}

	err = yaml.Unmarshal(yamlFile, &sources.List)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  filename,
			"Error": err,
		}).Fatal("Can not unmarshal YAML file")
	}

	os.MkdirAll("sources", os.ModePerm)

	for _, element := range sources.List {
		log.Infof("Processing URL:%s \n", element.URL)

		outputPath := filepath.Join(path, "sources", element.Name)
		downloadFile(outputPath, element.URL)
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
		cleanHTTP := strings.Replace(el, "http://", "", -1)
		cleanElement := strings.TrimSpace(cleanHTTP)

		isElementURL := govalidator.IsURL(cleanElement)
		isElementIP := govalidator.IsIP(cleanElement)
		startsWithHash := strings.HasPrefix(cleanLine, "#")

		if isElementURL == true && isElementIP == false && startsWithHash == false {
			validURL = cleanElement
		}
	}

	log.Debugf("Extracted URL:%s from line: %s\n", validURL, line)

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
		log.WithFields(log.Fields{
			"file":  openFile,
			"Error": err,
		}).Fatal("Can not process file")
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
		log.WithFields(log.Fields{
			"file":  openFile,
			"Error": err,
		}).Fatal("Can not process file")
	}

	log.WithFields(log.Fields{
		"File Path":         pathToFile,
		"Total Lines Found": linesInFile,
		"Lines Parsed":      linesParsed,
	}).Debug("File Parsing results")

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
		log.WithFields(log.Fields{
			"directory": folderPath,
			"Error":     err,
		}).Error("Can not process directory")
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

// HTTPResponse -XXX
type HTTPResponse struct {
	statusCode int
	URL        string
}

// DNSResponse -XXX
type DNSResponse struct {
	IPAddresses []string
	URL         string
}

func getURLDNSResponse(URL string, resultChan chan DNSResponse, wg *sync.WaitGroup) {
	result := DNSResponse{IPAddresses: []string{}, URL: URL}
	u, err := url.Parse(URL)
	if err != nil {
		log.Errorf("Invalid URL: %s", err)
	}

	ips, err := net.LookupIP(u.String())
	if err != nil {
		log.Errorf("Error parsing URL: %s", err)
	}
	for _, ip := range ips {
		result.IPAddresses = append(result.IPAddresses, ip.String())
	}

	resultChan <- result
}

func getURLStatusCode(URL string, resultChan chan HTTPResponse, wg *sync.WaitGroup) {
	result := HTTPResponse{statusCode: 0, URL: URL}
	u, err := url.Parse(URL)
	if err != nil {
		log.Errorf("Invalid URL: %s", err)
	}
	if len(u.Scheme) == 0 {
		u.Scheme = "http"
	}

	timeout := time.Duration(10 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	response, err := client.Get(u.String())

	if err != nil {
		log.Errorf("Error parsing URL: %s", err)
	}
	if response != nil {
		result = HTTPResponse{statusCode: response.StatusCode, URL: URL}
		defer response.Body.Close()
	}
	resultChan <- result
}

// Function for removing dead domains from a list
// :param domains: a list of domains
// Usage::
//   >>> workingDomains = cleanupDeadDomains(domains=['005.free-counter.co.uk', 'warning-0auto7.stream'])
//
func cleanupDeadDomains(domains []string) (result []HTTPResponse, err error) {

	resultChan := make(chan HTTPResponse, len(domains))
	var wg sync.WaitGroup

	for _, url := range domains {
		wg.Add(1)

		go func(url string) {
			getURLStatusCode(url, resultChan, &wg)
			defer wg.Done()
		}(url)

	}

	wg.Wait()
	close(resultChan)

	// resultChan = result
	for r := range resultChan {
		log.Infof("%s, %v", r.URL, r.statusCode)
	}

	// result = []string{"test"}

	return result, nil
}

// Function for removing domains with no DNS records from a list
// :param domains: a list of domains
// Usage::
//   >>> workingDomains = cleanupDomainsNoDNS(domains=['005.free-counter.co.uk', 'warning-0auto7.stream'])
//
func cleanupDomainsNoDNS(domains []string) (result []DNSResponse, err error) {

	resultChan := make(chan DNSResponse, len(domains))
	var wg sync.WaitGroup

	for _, url := range domains {
		wg.Add(1)

		go func(url string) {
			getURLDNSResponse(url, resultChan, &wg)
			defer wg.Done()
		}(url)

	}

	wg.Wait()
	close(resultChan)

	for r := range resultChan {
		// DNS Record found
		if len(r.IPAddresses) > 0 {
			result = append(result, r)
			log.Infof("Url: %s, IPs: %v", r.URL, r.IPAddresses)
		}

	}

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
func exportToFile(domains []string, format string, path string, ipAddress string) (err error) {
	if len(ipAddress) == 0 {
		ipAddress = "0.0.0.0"
	}
	return nil
}

func main() {

	start := time.Now()

	// absolutePathToFolder := filepath.Join(path, "sources")
	// absolutePathToFile := filepath.Join(path, "sources", "example")
	// hosts, _ := parseFile(absolutePathToFile)
	hosts := []string{}

	for i := 1; i <= 2; i++ {
		hosts = append(hosts, "google.com")
		hosts = append(hosts, "example.org")
		hosts = append(hosts, "something.xyx")
	}
	log.Info(len(hosts))
	// result, _ := cleanupDeadDomains(hosts)

	// Check DNS
	dnsCheck, _ := cleanupDomainsNoDNS(hosts)
	log.Debug(len(dnsCheck))

	// log.Debug(result)
	elapsed := time.Since(start)
	log.Printf("Executed in: %s", elapsed)
}
