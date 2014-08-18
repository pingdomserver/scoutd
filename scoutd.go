package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oguzbilgic/pusher"
	// "kylelemons.net/go/daemon"

	"github.com/scoutapp/scoutd/scoutd"
)

var config scoutd.ScoutConfig

func main() {
	scoutd.LoadConfig(&config) // load the yaml configuration into global struct 'config'
	log.Printf("Using Configuration: %#v\n", config)
	// dropPrivs() // change the effective UID/GID
	// configureLogger() // Create the logger interface, make sure we can log
	// changeToRunDir()

	if config.SubCommand == "config" {
		scoutd.GenConfig(config)
		os.Exit(0)
	}
	if config.SubCommand == "start" {
		log.Println("Starting daemon")
		startDaemon()
	}
	if config.SubCommand == "status" {
		log.Println("Checking status")
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
		panic(err)
	}

	commandChannel := conn.Channel(config.AccountKey + "-" + config.HostName)

	var agentRunning = &sync.Mutex{}
	fmt.Println("created agent")

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

func reportLoop(agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	c := time.Tick(60 * time.Second)
	for _ = range c {
		fmt.Println("report loop")
		checkin(agentRunning)
	}
	wg.Done()
}

func listenForRealtime(commandChannel **pusher.Channel, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("streamer_command") // a go channel is returned

	for {
		msg := <-messages
		cmdOpts := append(config.PassthroughOpts, "realtime", msg.(string))
		fmt.Printf("Running %s %s", config.AgentGemBin, strings.Join(cmdOpts, ""))
		cmd := exec.Command(config.AgentGemBin, cmdOpts...)
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

func listenForUpdates(commandChannel **pusher.Channel, agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("check_in") // a go channel is returned

	for {
		var _ = <-messages
		fmt.Println("got checkin command")
		checkin(agentRunning)
	}
}

func checkin(agentRunning *sync.Mutex) {
	fmt.Println("waiting on agent")
	agentRunning.Lock()
	cmdOpts := append(config.PassthroughOpts, config.AccountKey)

	fmt.Println("running agent: " + config.AgentGemBin + " " + strings.Join(config.PassthroughOpts, " ") + " " + config.AccountKey)
	cmd := exec.Command(config.AgentGemBin, cmdOpts...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent finished")
	agentRunning.Unlock()
	fmt.Println("agent available")
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

	// TODO:
	// can we write to log dir?
	// 
	rubyInfo, err := scoutd.CheckRubyEnv(config)
	if err != nil {
		log.Fatalf("Error checking Ruby env: %s", err)
	}
	for _, pathInfo := range rubyInfo {
		log.Println(pathInfo)
	}

	return nil
}
