package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/scoutapp/pusher"

	"github.com/scoutapp/scoutd/collectors"
	"github.com/scoutapp/scoutd/scoutd"
)

var config scoutd.ScoutConfig
var activeCollectors map[string]collectors.Collector

func main() {
	os.Setenv("SCOUTD_VERSION", scoutd.Version) // Used by child processes to determine if they are being run under scoutd
	log.SetPrefix("scoutd: ")                   // Set the default log prefix

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
	if config.SubCommand == "test" {
		config.Log.Println("Testing plugin")
	}
}

func startDaemon() {
	// Sleep before startup.
	// Just precautionary so that we don't consume 100% CPU in case of respawn loops
	time.Sleep(1 * time.Second)

	// All necessary configuration checks and setup tasks should pass
	// Just log the error for now
	if err := sanityCheck(); err != nil {
		config.Log.Printf("Error: %s", err)
	}

	var wg sync.WaitGroup
	wg.Add(1) // end the program if any loops finish (they shouldn't)

	var agentRunning = &sync.Mutex{}
	config.Log.Println("Created agent")

	go initCollectors()
	go initPayloadEndpoint()
	go initPusher(agentRunning, &wg)
	go reportLoop(agentRunning, &wg)

	wg.Wait()
}

// Initialize and start Collectors
// Hardcoded to start a single statsdCollector for now.
func initCollectors() {
	activeCollectors = make(map[string]collectors.Collector)

	if config.Statsd.Enabled == "true" {
		flushInterval := time.Duration(60) * time.Second
		if statsd, err := collectors.NewStatsdCollector("statsd", config.Statsd.Addr, flushInterval, collectors.DefaultEventLimit); err != nil {
			config.Log.Printf("error creating statsd collector: %s", err)
		} else {
			statsd.Start()
			activeCollectors[statsd.Name()] = statsd
		}
	}
}

// The Ruby scout-client will be fetching json data from the Scout Collectors and
// including that in the checkin bundle.
func initPayloadEndpoint() {
	http.HandleFunc("/", writePayload)
	http.ListenAndServe(scoutd.DefaultPayloadAddr, nil)
}

// Compiles the Collector.Payload() data and encodes to json and writes to w.
func writePayload(w http.ResponseWriter, r *http.Request) {
	payloads := make([]*collectors.CollectorPayload, len(activeCollectors))
	i := 0
	for _, c := range activeCollectors {
		payloads[i] = c.Payload()
		i++
	}
	p := make(map[string][]*collectors.CollectorPayload, 1)
	p["collectors"] = payloads
	js, err := json.Marshal(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func initPusher(agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	var conn *pusher.Connection
	var err error
	for ; ; time.Sleep(30 * time.Second) {
		if conn == nil {
			config.Log.Println("Connecting to Pusher")
			conn, err = pusher.New("f07eaa39898f3c36c8cf")
			if err != nil {
				config.Log.Printf("Error connecting to pusher: %s", err)
			} else {
				config.Log.Println("Connected to Pusher")
				commandChannel := conn.Channel(config.AccountKey + "-" + config.HostName)
				if commandChannel == nil {
					config.Log.Printf("Error creating pusher channel: %s", err)
				} else {
					go listenForRealtime(&commandChannel, wg)
					go listenForUpdates(&commandChannel, agentRunning, wg)
				}
			}
		}
	}
	wg.Done()
}

func checkStatus() {
	// scoutd.checkPidRunning(cfg.PidFile) // Check the PID file to see if scout is running
	sanityCheck()
}

func runDebug() {
	stringDivider := "#####################"
	config.Log.Printf("\n\n%s\nRunning scout debug\n%s\n\n", stringDivider, stringDivider)
	config.Log.Printf("\n\nCurrent scoutd configuration:\n%#v\n\n", config)
	config.Log.Printf("\n\nRunning `scout troubleshoot`\n%s\n\n", stringDivider)
	cmd := exec.Command(config.RubyPath, append([]string{config.AgentRubyBin}, "troubleshoot")...)
	if out, err := cmd.Output(); err != nil {
		config.Log.Printf("Error running agent: %s", err)
	} else {
		config.Log.Printf("\n%s\n", out)
	}
	config.Log.Printf("\n\n%s\nEnd scout troubleshoot\n%s\n\n", stringDivider, stringDivider)
}

func reportLoop(agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	time.Sleep(2 * time.Second)      // Sleep 2 seconds after initial startup
	checkin(agentRunning, true)      // Initial checkin - use forceCheckin=true
	time.Sleep(scoutd.DurationToNextMinute() * time.Second) // Start regular checkin interval at the beginning of every minute
	config.Log.Println("Report loop")
	checkin(agentRunning, false)
	c := time.Tick(60 * time.Second) // Fire precisely every 60 seconds from now on
	for _ = range c {
		config.Log.Println("Report loop")
		checkin(agentRunning, false)
	}
	wg.Done()
}

func listenForRealtime(commandChannel **pusher.Channel, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("streamer_command") // a go channel is returned

	var rtReadPipe, rtWritePipe *os.File // We'll use these to store the pointers to the current pipes for realtime
	var cmdOutput []byte
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
					var execPath string  // This will be the initial program to invoke - either "nice" or config.RubyPath
					var cmdOpts []string // The slice of all options when running execPath
					var nicePath string
					nicePath, err = exec.LookPath("nice")
					if err != nil {
						execPath = config.RubyPath             // No "nice" program found, run config.RubyPath directly
						config.Log.Printf("Notice: %s\n", err) // Log a notice about not using "nice"
					} else {
						execPath = nicePath                                    // Run the realtime ruby through "nice"
						cmdOpts = append(cmdOpts, "-n", "10", config.RubyPath) // set nice level to 10 when invoking config.RubyPath
					}
					cmdOpts = append(cmdOpts, config.AgentRubyBin)
					cmdOpts = append(cmdOpts, config.PassthroughOpts...)
					cmdOpts = append(cmdOpts, "realtime", msg.(string))
					config.Log.Printf("Running %s %s ExtraFiles: %#v", execPath, strings.Join(cmdOpts, " "), []*os.File{rtReadPipe})
					rtCmd := exec.Command(execPath, cmdOpts...)
					rtCmd.ExtraFiles = []*os.File{rtReadPipe} // Pass the reading pipe handle to the agent as fd 3. http://golang.org/pkg/os/exec/#Cmd
					cmdOutput, err = rtCmd.CombinedOutput()
					if err != nil {
						config.Log.Printf("Error running realtime: %#v", err)
						config.Log.Printf("Agent output: %s\n", cmdOutput)
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
		checkin(agentRunning, true)
	}
}

func checkin(agentRunning *sync.Mutex, forceCheckin bool) {
	os.Setenv("SCOUTD_PAYLOAD_URL", fmt.Sprintf("http://%s/", scoutd.DefaultPayloadAddr))
	config.Log.Println("Waiting on agent")
	agentRunning.Lock()
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

	rubyPath, err := scoutd.GetRubyPath(config.RubyPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error checking Ruby path: %s\n", err))
	}
	config.Log.Printf("Found Ruby at path: %s\n", rubyPath)

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
