package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/oguzbilgic/pusher"
	// "kylelemons.net/go/daemon"

	"github.com/scoutapp/scoutd/scoutd"
)

var config scoutd.ScoutConfig

func main() {
	log.SetPrefix("scoutd: ") // Set the default log prefix

	scoutd.LoadConfig(&config) // load the yaml configuration into global struct 'config'

	// Try to change to config.RunDir, if specified.
	// Fatal if we cannot change to the directory
	if config.RunDir != "" {
		if err := os.Chdir(config.RunDir); err != nil {
			config.Log.Fatalf("Unable to change to RunDir: %s", err)
		}
	}

	// Listen for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGUSR1)
	go signalHandler(sigChan)

	// What command was invoked
	if config.SubCommand == "config" {
		scoutd.GenConfig(config)
		os.Exit(0)
	}
	if config.SubCommand == "start" {
		config.Log.Println("Starting daemon")
		startDaemon()
	}
	if config.SubCommand == "status" {
		config.Log.Println("Checking status")
		checkStatus()
	}
}

func startDaemon() {
	os.Setenv("GEM_PATH", fmt.Sprintf("%s:%s", config.GemPath, ":", os.Getenv("GEM_PATH"))) // Prepend the GEM_PATH
	sanityCheck() // All necessary configuration checks and setup tasks must pass, otherwise sanityCheck will cause us to exit

	var wg sync.WaitGroup
	wg.Add(1) // end the program if any loops finish (they shouldn't)

	conn, err := pusher.New("f07eaa39898f3c36c8cf")
	if err != nil {
		config.Fatalf("Error creating pusher channel: %s", err)
	}

	commandChannel := conn.Channel(config.AccountKey + "-" + config.HostName)

	var agentRunning = &sync.Mutex{}
	config.Log.Println("Created agent")

	go listenForRealtime(&commandChannel, &wg)
	go reportLoop(agentRunning, &wg)
	go listenForUpdates(&commandChannel, agentRunning, &wg)
	wg.Wait()
	// daemon.Run() // daemonize
}

func checkStatus() {
	// scoutd.checkPidRunning(cfg.PidFile) // Check the PID file to see if scout is running
	sanityCheck()
}

func runDebug() {
	// Run scout troubleshoot, etc.
	config.Log.Printf("Running scout troubleshoot.\n")
}

func reportLoop(agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	c := time.Tick(60 * time.Second)
	for _ = range c {
		config.Log.Println("Report loop")
		checkin(agentRunning)
	}
	wg.Done()
}

func listenForRealtime(commandChannel **pusher.Channel, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("streamer_command") // a go channel is returned

	for {
		msg := <-messages
		cmdOpts := append(config.PassthroughOpts, "realtime", msg.(string))
		config.Log.Printf("Running %s %s", config.AgentGemBin, strings.Join(cmdOpts, ""))
		cmd := exec.Command(config.AgentGemBin, cmdOpts...)
		err := cmd.Run()
		if err != nil {
			config.Log.Fatal(err)
		}
	}
	wg.Done()
}

func listenForUpdates(commandChannel **pusher.Channel, agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("check_in") // a go channel is returned

	for {
		var _ = <-messages
		config.Log.Println("Got check_in command")
		checkin(agentRunning)
	}
}

func checkin(agentRunning *sync.Mutex) {
	config.Log.Println("Waiting on agent")
	agentRunning.Lock()
	cmdOpts := append(config.PassthroughOpts, config.AccountKey)

	config.Log.Println("Running agent: " + config.AgentGemBin + " " + strings.Join(config.PassthroughOpts, " ") + " " + config.AccountKey)
	cmd := exec.Command(config.AgentGemBin, cmdOpts...)
	err := cmd.Run()
	if err != nil {
		config.Log.Fatal(err)
	}
	config.Log.Println("Agent finished")
	agentRunning.Unlock()
	config.Log.Println("Agent available")
}

func sanityCheck() error {
	if config.AccountKey == "" {
		return errors.New("Account key is not configured! Scout will not be able to check in.")
	} else {
		keyIsValid, err := scoutd.AccountKeyValid(config.AccountKey, "", config.HttpClients.HttpClient) // Make sure the account key is the correct format, and optionally verify against the reportingServerUrl
		if err != nil {
			return err
		} else if !keyIsValid {
			return errors.New(fmt.Sprintf("Invalid account key: %s", config.AccountKey))
		}
	}

	rubyInfo, err := scoutd.CheckRubyEnv(config)
	if err != nil {
		config.Log.Fatalf("Error checking Ruby env: %s", err)
	}
	for _, pathInfo := range rubyInfo {
		config.Log.Println(pathInfo)
	}

	return nil
}

func signalHandler(sigChan <-chan os.Signal) {
	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			config.Log.Printf("Received SIGHUP. Reloading configuration.\n")
			scoutd.LoadConfig(&config)
		case syscall.SIGUSR1:
			config.Log.Printf("Received SIGUSR1. Running debug/troublehsoot routine.\n")
			runDebug()
		}
	}
}