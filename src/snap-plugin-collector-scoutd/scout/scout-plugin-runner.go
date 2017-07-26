package scout

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"scoutd"
	"strings"
)

func RunScout() error {
	config := loadConfiguration()
	checkin(true, config)
	return nil
}

func loadConfiguration() *scoutd.ScoutConfig {
	var config scoutd.ScoutConfig
	scoutd.LoadConfig(&config)

	// Try to change to config.RunDir, if specified.
	// Fatal if we cannot change to the directory
	if config.RunDir != "" {
		if err := os.Chdir(config.RunDir); err != nil {
			config.Log.Fatalf("Unable to change to RunDir: %s", err)
		}
	}
	config.Log.Printf("Using configuration: %v\n", config)
	config.Log.Printf("Starting scout-client")

	// All necessary configuration checks and setup tasks should pass
	// Just log the error for now
	if err := sanityCheck(config); err != nil {
		config.Log.Printf("Error: %s", err)
	}
	return &config
}

func checkin(forceCheckin bool, config *scoutd.ScoutConfig) {
	os.Setenv("SCOUTD_PAYLOAD_URL", fmt.Sprintf("http://%s/", scoutd.DefaultPayloadAddr))
	cmdOpts := append([]string{config.AgentRubyBin}, config.PassthroughOpts...)
	if forceCheckin {
		cmdOpts = append(cmdOpts, "-F")
	}
	cmdOpts = append(cmdOpts, config.AccountKey)
	config.Log.Printf("Running agent: %s %s\n", config.RubyPath, strings.Join(cmdOpts, " "))
	cmd := exec.Command(config.RubyPath, cmdOpts...)

	if cmdOutput, err := cmd.CombinedOutput(); err != nil {
		config.Log.Printf("Error running agent: %s", err)
		config.Log.Printf("Agent output: \n%s", cmdOutput)
	} else {
		var checkinData scoutd.AgentCheckin
		scanner := bufio.NewScanner(bytes.NewReader(cmdOutput))
		for scanner.Scan() {
			err := json.Unmarshal(scanner.Bytes(), &checkinData)
			if err == nil {
				break
			}
		}
		if checkinData.Success == true {
			config.Log.Println("Agent successfully checked in.")
			if config.LogLevel != "" {
				config.Log.Printf("Agent output: \n%s", cmdOutput)
			}
		} else {
			config.Log.Printf("Error: Agent was not able to check in. Server response: %#v", checkinData.ServerResponse)
			config.Log.Printf("Agent output: \n%s", cmdOutput)
		}
	}
	config.Log.Println("Agent finished")
}

func sanityCheck(config scoutd.ScoutConfig) error {
	if config.AccountKey == "" {
		return errors.New("Account key is not configured! Scout will not be able to check in.")
	} else {
		// Make sure the account key is the correct format, and verify against the reportingServerUrl
		keyIsValid, err := scoutd.AccountKeyValid(config)
		if err != nil {
			return err
		} else if !keyIsValid {
			return errors.New(fmt.Sprintf("Invalid account key: %s", config.AccountKey))
		}
	}

	rubyPath, err := scoutd.GetRubyPath(config.RubyPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error checking Ruby path: %s\n", err))
	}
	config.Log.Printf("Found Ruby at path: %s\n", rubyPath)

	return nil
}
