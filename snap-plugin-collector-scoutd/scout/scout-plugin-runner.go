package scout

import (
	// "bufio"
	// "bytes"
	// "encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pingdomserver/mergo"
	"github.com/pingdomserver/scoutd/scoutd"
)

func RunScout() ([]byte, error) {
	configFilePath := scoutd.LoadDefaults().ConfigFile
	config, error := loadConfiguration(configFilePath)
	if error != nil {
		return nil, error
	}
	cmdData, err := checkin(true, config)
	if err != nil {
		return nil, err
	}
	return cmdData, nil
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
	// append -j (json) option to configuration
	cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-j")

	log.Printf("Using configuration: %v\n", cfg)

	return &cfg, nil
}

func checkin(forceCheckin bool, config *scoutd.ScoutConfig) ([]byte, error) {
	// Try to change to config.RunDir, if specified.
	// Fatal if we cannot change to the directory
	if config.RunDir != "" {
		if err := os.Chdir(config.RunDir); err != nil {
			config.Log.Fatalf("Unable to change to RunDir: %s", err)
			return nil, errors.New("Unable to change to RunDir.")
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
	log.Printf("majonez")
	if cmdOutput, err := cmd.CombinedOutput(); err != nil {
		config.Log.Printf("Error running agent: %s", err)
		config.Log.Printf("Agent output: \n%s", cmdOutput)
		return cmdOutput, errors.New("Error running agent.")
	} else {
		log.Printf("MUCHOMIOR %s", cmdOutput)
		return cmdOutput, err
	}
	config.Log.Println("Agent finished")

	return nil, nil
}
