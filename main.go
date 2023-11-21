package main

import (
	"bufio"

	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"sort"
	"strings"
	"time"
)

const (
	version     = "0.0.1"
	releaseDate = "21/11/23"
	projectURL  = "swapless/aeigo"
	blockIP     = "0.0.0.0"
)

var (
	inputHosts  = "/etc/hosts"
	outputHosts = "/etc/hosts"
)

var (
	autorun   bool
	restore   bool
	debug     bool
	uninstall bool
)

func main() {
	downloadAndBuildBlocklists()
}

func downloadAndBuildBlocklists() {
	var blacklistSources []string
	var whitelistSources []string

	blacklistFile := "./lists/blacklist.sources"
	whitelistFile := "./lists/whitelist.sources"

	blacklistSources = readSourcesFromFile(blacklistFile, blacklistSources)
	whitelistSources = readSourcesFromFile(whitelistFile, whitelistSources)

	blacklistDomains := downloadAndMergeSources(blacklistSources)
	whitelistDomains := downloadAndMergeSources(whitelistSources)

	extractedBlacklistDomains := extractDomains(blacklistDomains)
	extractedWhitelistDomains := extractDomains(whitelistDomains)

	finalHosts := buildFinalHostsFile(extractedBlacklistDomains, extractedWhitelistDomains)

	// ! Bug : Modifies /etc/hosts multiple times with the init header
	writeToFile(outputHosts, finalHosts)

	websitesBlockedCounter := countBlockedWebsites(finalHosts)

	fmt.Printf("\ndone, %d websites blocked.\n\n", websitesBlockedCounter)
}

func readSourcesFromFile(filepath string, sources []string) []string {
	if _, err := os.Stat(filepath); err == nil {
		file, err := os.Open(filepath)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return sources
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			source := strings.TrimSpace(scanner.Text())
			if len(source) > 0 {
				sources = append(sources, source)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error scanning file:", err)
		}
	}
	return sources
}

func downloadAndMergeSources(sources []string) string {
	var mergedDomains strings.Builder

	for _, source := range sources {
		downloadedDomains := downloadFile(source)
		mergedDomains.WriteString(downloadedDomains)
	}

	return mergedDomains.String()
}

func downloadFile(url string) string {
	fmt.Printf("Downloading %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading:", err)
		return ""
	}
	defer resp.Body.Close()

	var domains strings.Builder
	io.Copy(&domains, resp.Body)

	return domains.String()
}

func extractDomains(domains string) []string {
	domainsSet := make(map[string]bool)
	lines := strings.Split(domains, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			domainsSet[line] = true
		}
	}

	var extracted []string
	for domain := range domainsSet {
		extracted = append(extracted, domain)
	}

	sort.Strings(extracted)
	return extracted
}

func buildFinalHostsFile(blacklist []string, whitelist []string) string {
	var finalHosts strings.Builder

	// Append original user hosts or comments if not found
	userHostsContent, err := ioutil.ReadFile(inputHosts)
	if err != nil {
		finalHosts.WriteString("# Original user hosts not found or couldn't be read\n")
	} else {
		finalHosts.Write(userHostsContent)
	}

	// Append generated comments for ad-blocking
	finalHosts.WriteString("\n# Ad blocking hosts generated " + time.Now().Format(time.RFC3339) + "\n")
	finalHosts.WriteString("# Don't write below this line. It will be lost if you run aeigo again.\n\n")

	// Append filtered blacklist entries with the block IP
	for _, domain := range blacklist {
		finalHosts.WriteString(blockIP + " " + domain + "\n")
	}

	return finalHosts.String()
}

func countBlockedWebsites(hostsContent string) int {
	lines := strings.Split(hostsContent, "\n")
	counter := 0

	for _, line := range lines {
		if strings.HasPrefix(line, blockIP) {
			counter++
		}
	}

	return counter
}

func writeToFile(filepath string, content string) {
	err := ioutil.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

func getLineCount(pattern, filename string) int {
	file, err := os.Open(filename)
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			return count
		}
		count++
	}
	return -1
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func copyFirstNLines(src, dst string, n int) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	scanner := bufio.NewScanner(source)
	lines := 0
	for scanner.Scan() {
		if lines >= n {
			break
		}
		_, err := destination.WriteString(scanner.Text() + "\n")
		if err != nil {
			return err
		}
		lines++
	}
	return nil
}

func removeEmptyLines(inputFile, outputFile string) error {
	source, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) > 0 {
			_, err := destination.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}
