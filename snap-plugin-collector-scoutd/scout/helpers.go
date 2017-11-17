package scout

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func AccountKeyValid(config ScoutConfig) (bool, error) {
	serverUrl := config.ReportingServerUrl
	if serverUrl == "" {
		serverUrl = DefaultScoutUrl
	}
	// Check the format of the account key - 40 chars, 0-9A-Za-z
	matched, err := regexp.MatchString("^[0-9A-Za-z]{40}$", config.AccountKey)
	if err != nil || !matched {
		return false, err
	} else if matched {
		var client *http.Client
		// Select the correct transport based on the URL
		if strings.HasPrefix(serverUrl, "https://") {
			client = config.HttpClients.HttpsClient
		} else {
			client = config.HttpClients.HttpClient
		}
		postUrl := serverUrl + fmt.Sprintf("/account/%s/valid.scout", config.AccountKey)
		resp, err := client.Get(postUrl)
		if err != nil {
			return false, err
		} else if resp.StatusCode == 200 {
			return true, nil
		}
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

func GetRubyPath(checkPath string) (string, error) {
	var rubyPath string

	if checkPath != "" {
		path, err := exec.LookPath(checkPath)
		if err != nil {
			return "", err
		}
		rubyPath = path
	} else {
		path, err := exec.LookPath("ruby")
		if err != nil {
			return "", err
		}
		rubyPath = path
	}
	return rubyPath, nil
}


func DurationToNextMinute() time.Duration {
	return time.Duration(60 - time.Now().Second())
}
