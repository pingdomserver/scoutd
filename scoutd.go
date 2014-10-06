package main

import (
	"encoding/json"
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
	if config.SubCommand == "debug" {
		config.Log.Println("Running debug")
		runDebug()
	}
}

func startDaemon() {
	// Sleep before startup.
	// Just precautionary so that we don't consume 100% CPU in case of respawn loops
	time.Sleep(1 * time.Second)

	// Prepend the GEM_PATH
	os.Setenv("GEM_PATH", fmt.Sprintf("%s:%s", config.GemPath, ":", os.Getenv("GEM_PATH")))

	// All necessary configuration checks and setup tasks should pass
	// Just log the error for now
	if err := sanityCheck(); err != nil {
		config.Log.Printf("Error: %s", err)
	}

	var wg sync.WaitGroup
	wg.Add(1) // end the program if any loops finish (they shouldn't)

	conn, err := pusher.New("f07eaa39898f3c36c8cf")
	if err != nil {
		config.Log.Fatalf("Error creating pusher channel: %s", err)
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
	os.Setenv("GEM_PATH", fmt.Sprintf("%s:%s", config.GemPath, ":", os.Getenv("GEM_PATH")))
	stringDivider := "#####################"
	config.Log.Printf("\n\n%s\nRunning scout debug\n%s\n\n", stringDivider, stringDivider)
	config.Log.Printf("\n\nCurrent scoutd configuration:\n%#v\n\n", config)
	config.Log.Printf("\n\nRunning `scout troubleshoot`\n%s\n\n", stringDivider)
	cmd := exec.Command(config.AgentGemBin, "troubleshoot")
	if out, err := cmd.Output(); err != nil {
		config.Log.Printf("Error running agent: %s", err)
	} else {
		config.Log.Printf("\n%s\n", out)
	}
	config.Log.Printf("\n\n%s\nEnd scout troubleshoot\n%s\n\n", stringDivider, stringDivider)
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

	var rtReadPipe, rtWritePipe *os.File // We'll use these to store the pointers to the current pipes for realtime
	var err error

	var rtExit = make(chan int, 1) // Communication channel to know when realtime has exited
	var rtRunning = false          // Tracks whether realtime is running or not
	for {
		select {
		case <-rtExit:
			config.Log.Println("Realtime exited.")
			rtRunning = false
		case msg := <-messages: // We received a message from pusher
			config.Log.Printf("Got pusher message: %#v\n", msg)
			if rtRunning == false { // Realtime is not running
				config.Log.Printf("Spawning realtime\n")
				rtRunning = true // Mark realtime as running
				rtReadPipe, rtWritePipe, err = os.Pipe()
				if err != nil { // Create new pipes for communicating to realtime
					config.Log.Fatal(err)
				}
				go func() {
					cmdOpts := append(config.PassthroughOpts, "realtime", msg.(string))
					config.Log.Printf("Running %s %s ExtraFiles: %#v", config.AgentGemBin, strings.Join(cmdOpts, " "), []*os.File{rtReadPipe})
					rtCmd := exec.Command(config.AgentGemBin, cmdOpts...)
					rtCmd.ExtraFiles = []*os.File{rtReadPipe} // Pass the reading pipe handle to the agent as fd 3. http://golang.org/pkg/os/exec/#Cmd
					err := rtCmd.Run()
					if err != nil {
						config.Log.Printf("Error running realtime: %#v", err)
					}
					config.Log.Println("Done waiting for realtime")
					rtReadPipe.Close()
					rtWritePipe.Close()
					rtExit <- 1
				}()
			} else {
				config.Log.Printf("Realtime is running. Writing message to pipe.\n")
				rtWritePipe.Write([]byte(msg.(string))) // Convert msg to string, then put that in a byte array for writing
				config.Log.Printf("Done writing message to pipe\n")
			}
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

	config.Log.Printf("Running agent: %s %s %s\n", config.AgentGemBin, strings.Join(config.PassthroughOpts, " "), config.AccountKey)
	cmd := exec.Command(config.AgentGemBin, cmdOpts...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		config.Log.Printf("Error configuring StdoutPipe: %s", err)
	}
	if err := cmd.Start(); err != nil {
		config.Log.Fatalf("Error running agent: %s", err)
	}

	// Read stdout into json decoder
	var checkinData scoutd.AgentCheckin
	var stdoutErr error = nil
	for stdoutErr == nil {
		stdoutErr = json.NewDecoder(stdout).Decode(&checkinData)
		if stdoutErr != nil && stdoutErr.Error() != "EOF" {
			config.Log.Printf("Err from JSON decoder: %#v", stdoutErr)
		}
	}
	if err := cmd.Wait(); err != nil {
		config.Log.Printf("Err from Wait: %#v", err)
	}
	if checkinData.Success == true {
		config.Log.Println("Agent successfully checked in.")
	} else {
		config.Log.Printf("Error: Agent was not able to check in. Server response: %#v", checkinData.ServerResponse)
	}
	config.Log.Println("Agent finished")
	agentRunning.Unlock()
	config.Log.Println("Agent available")
}

func sanityCheck() error {
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
