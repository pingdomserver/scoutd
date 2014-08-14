package scoutd

import (
	"log"
	"os"
	"regexp"
	"strings"
)

func AccountKeyValid(key string, checkServer bool) (bool, error) {
	// Check the format of the account key - 40 chars, 0-9A-Za-z
	matched, err := regexp.MatchString("^[0-9A-Za-z]{40}$", key)
	if err != nil || !matched {
		return false, err
	}
	return true, err
}

func ShortHostname() string {
	var hostname, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(hostname, ".")[0]
}