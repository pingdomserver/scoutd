package scoutd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func AccountKeyValid(key string, serverUrl string, client *http.Client) (bool, error) {
	// Check the format of the account key - 40 chars, 0-9A-Za-z
	matched, err := regexp.MatchString("^[0-9A-Za-z]{40}$", key)
	if err != nil || !matched {
		return false, err
	} else if matched && serverUrl != "" {
		json := fmt.Sprintf("{\"key\":\"%s\"}", key)
		b := strings.NewReader(json)
		resp, err := client.Post(serverUrl, "application/json", b)
		if err != nil {
			return false, err
		} else if resp.StatusCode == 200 {
			return true, nil
		}
	} else if matched && serverUrl == "" {
		return true, nil
	}
	return false, nil
}

func ShortHostname() string {
	var hostname, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(hostname, ".")[0]
}

func CheckRubyEnv(config ScoutConfig) ([]string, error) {
	var rubyPaths []string
	var err error
	var path string
	path, err = exec.LookPath("ruby")
	if err != nil {
		return rubyPaths, err
	}
	rubyPaths = append(rubyPaths, fmt.Sprintf("Ruby binary found at %s", path))
	path, err = exec.LookPath("gem")
	if err != nil {
		return rubyPaths, err
	}
	rubyPaths = append(rubyPaths, fmt.Sprintf("Gem binary found at %s", path))
	return rubyPaths, nil
}