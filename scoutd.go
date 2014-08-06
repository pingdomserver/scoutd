package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/oguzbilgic/pusher"
	// "kylelemons.net/go/daemon"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	accountKey := os.Getenv("SCOUT_KEY")
	scoutGemBinPath := os.Getenv("SCOUT_GEM_BIN_PATH") + "/scout"

	conn, err := pusher.New("f07eaa39898f3c36c8cf")
	if err != nil {
		panic(err)
	}

	hostName := os.Getenv("SCOUT_HOSTNAME")
	commandChannel := conn.Channel(accountKey + "-" + hostName)

	agentRunning := make(chan bool)

	go listenForRealtime(&commandChannel, scoutGemBinPath, &wg)
	go reportLoop(accountKey, scoutGemBinPath, &wg)
	go listenForUpdates(scoutGemBinPath, accountKey, &commandChannel, &wg)
	wg.Wait()
	// daemon.Run() // daemonize
}

func reportLoop(accountKey string, scoutGemBinPath string, wg *sync.WaitGroup) {
	c := time.Tick(1 * time.Second)
	for _ = range c {
		// fmt.Printf("report loop\n")
		initiateCheckin(scoutGemBinPath, accountKey)
	}
	wg.Done()
}

func listenForRealtime(commandChannel **pusher.Channel, scoutGemBinPath string, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("streamer_command") // a go channel is returned

	for {
		msg := <-messages
		fmt.Printf(scoutGemBinPath + " realtime " + msg.(string))
		cmd := exec.Command(scoutGemBinPath, "realtime", msg.(string))
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

func listenForUpdates(scoutGemBinPath string, accountKey string, commandChannel **pusher.Channel, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("update_command") // a go channel is returned

	for {
		msg := <-messages
		fmt.Printf(msg.(string))
		for {
			running := <-agentRunning
			if running == false {
				initiateCheckin(scoutGemBinPath, accountKey)
				break
			}
		}
	}
}

func initiateCheckin(scoutGemBinPath string, accountKey string) {
	cmd := exec.Command(scoutGemBinPath, accountKey)
	agentRunning <- true
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	agentRunning <- false
}
