package scout

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/scoutserver/mergo"
	"log"
	"os"
	"os/exec"
	"scoutd"
	"strings"
)

func RunScout() error {
	configFilePath := scoutd.LoadDefaults().ConfigFile
	config, error := loadConfiguration(configFilePath)
	if error != nil {
		return error
	}
	return checkin(true, config)
}

func loadConfiguration(configFile string) (*scoutd.ScoutConfig, error) {
	cfg := scoutd.LoadDefaults()
	ymlOpts := scoutd.LoadConfigFile(configFile)
	if err := mergo.Merge(&cfg, ymlOpts); err != nil {
		log.Fatalf("Error while merging YAML file config options: %s\n", err)
		return nil, errors.New("Error while merging YAML file config options.")
	}
	scoutd.ConfigureLogger(&cfg)

	// Compile the passthroughOpts the scout ruby agent will need
	cfg.PassthroughOpts = make([]string, 0) // Make sure we reset to an empty array in case we are reloading the config
	cfg.PassthroughOpts = append(cfg.PassthroughOpts, "--hostname", cfg.HostName)
	if cfg.AgentEnv != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-e", cfg.AgentEnv)
	}
	if cfg.AgentRoles != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-r", cfg.AgentRoles)
	}
	if cfg.AgentDisplayName != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-n", cfg.AgentDisplayName)
	}
	if cfg.ReportingServerUrl != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-s", cfg.ReportingServerUrl)
	}
	if cfg.AgentDataFile != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-d", cfg.AgentDataFile)
	}
	if cfg.HttpProxyUrl != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "--http-proxy", cfg.HttpProxyUrl)
	}
	if cfg.HttpsProxyUrl != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "--https-proxy", cfg.HttpsProxyUrl)
	}
	if cfg.LogLevel != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-v", "-l", "debug")
	}

	if cfg.RubyPath == "" {
		cfg.RubyPath, _ = scoutd.GetRubyPath("")
	}
	log.Printf("Using configuration: %v\n", cfg)

	return &cfg, nil
}

func checkin(forceCheckin bool, config *scoutd.ScoutConfig) error {
	// Try to change to config.RunDir, if specified.
	// Fatal if we cannot change to the directory
	if config.RunDir != "" {
		if err := os.Chdir(config.RunDir); err != nil {
			config.Log.Fatalf("Unable to change to RunDir: %s", err)
			return errors.New("Unable to change to RunDir.")
		}
	}

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
		return errors.New("Error running agent.")
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
			return errors.New("Error: Agent was not able to check in.")
		}
	}
	config.Log.Println("Agent finished")
	return nil
}
