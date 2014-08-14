package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oguzbilgic/pusher"
	// "kylelemons.net/go/daemon"

	"./scoutd"
)

var config scoutd.ScoutConfig

func main() {
	var wg sync.WaitGroup
	wg.Add(1) // end the program if any loops finish (they shouldn't)

	scoutd.LoadConfig(&config) // load the yaml configuration into global struct 'config'

	sanityCheck() // All necessary configuration checks and setup tasks must pass, otherwise sanityCheck will cause us to exit

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
		fmt.Printf(config.AgentGemBin + " realtime " + msg.(string))
		cmd := exec.Command(config.AgentGemBin, "realtime", msg.(string))
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

func sanityCheck() {
	// dropPrivs() // change the effective UID/GID
	// configureLogger() // Create the logger interface, make sure we can log
	// changeToRunDir()
	keyIsValid, err := scoutd.AccountKeyValid(config.AccountKey, false) // Make sure the account key is the correct format, and optionally verify against the reportingServerUrl
	if err != nil {
		log.Fatal(err)
	} else if !keyIsValid {
		log.Fatalf("Invalid account key: %s\n", config.AccountKey)
	}
}
