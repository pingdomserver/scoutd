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
	wg.Add(1) // end the program if any loops finish (they shouldn't)
	accountKey := os.Getenv("SCOUT_KEY")
	scoutGemBinPath := os.Getenv("SCOUT_GEM_BIN_PATH") + "/scout"

	conn, err := pusher.New("f07eaa39898f3c36c8cf")
	if err != nil {
		panic(err)
	}

	hostName := os.Getenv("SCOUT_HOSTNAME")
	commandChannel := conn.Channel(accountKey + "-" + hostName)

	var agentRunning = &sync.Mutex{}
	fmt.Println("created agent")

	go listenForRealtime(&commandChannel, scoutGemBinPath, &wg)
	go reportLoop(accountKey, scoutGemBinPath, agentRunning, &wg)
	go listenForUpdates(scoutGemBinPath, accountKey, &commandChannel, agentRunning, &wg)
	wg.Wait()
	// daemon.Run() // daemonize
}

func reportLoop(accountKey, scoutGemBinPath string, agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	c := time.Tick(60 * time.Second)
	for _ = range c {
		fmt.Println("report loop")
		checkin(scoutGemBinPath, accountKey, agentRunning)
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

func listenForUpdates(scoutGemBinPath string, accountKey string, commandChannel **pusher.Channel, agentRunning *sync.Mutex, wg *sync.WaitGroup) {
	messages := commandChannel.Bind("check_in") // a go channel is returned

	for {
		var _ = <-messages
		fmt.Println("got checkin command")
		checkin(scoutGemBinPath, accountKey, agentRunning)
	}
}

func checkin(scoutGemBinPath string, accountKey string, agentRunning *sync.Mutex) {
	fmt.Println("waiting on agent")
	agentRunning.Lock()
	fmt.Println("running agent")
	cmd := exec.Command(scoutGemBinPath, accountKey)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent finished")
	agentRunning.Unlock()
	fmt.Println("agent available")
}
