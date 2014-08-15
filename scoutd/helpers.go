package scoutd

import (
	"fmt"
	"log"
	"net/http"
	"os"
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