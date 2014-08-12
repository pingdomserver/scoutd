package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oguzbilgic/pusher"
	"github.com/kylelemons/go-gypsy/yaml"
	// "kylelemons.net/go/daemon"
)

type Config struct {
	configFile string
	accountKey string
	hostName string
	userName string
	groupName string
	runDir string
	logDir string
	gemPath string
	gemBinPath string
	scoutGemBin string
	agentEnv string
	agentRoles string
	agentDataFile string
	httpProxyUrl string
	httpsProxyUrl string
	reportingServerUrl string
	passthroughOpts []string
}

var config Config

func main() {
	var wg sync.WaitGroup
	wg.Add(1) // end the program if any loops finish (they shouldn't)

	loadConfig(&config) // load the yaml configuration into global struct 'config' 

	conn, err := pusher.New("f07eaa39898f3c36c8cf")
	if err != nil {
		panic(err)
	}

	commandChannel := conn.Channel(config.accountKey + "-" + config.hostName)

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
		fmt.Printf(config.scoutGemBin + " realtime " + msg.(string))
		cmd := exec.Command(config.scoutGemBin, "realtime", msg.(string))
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
	cmdOpts := append(config.passthroughOpts, config.accountKey)

	fmt.Println("running agent: " + config.scoutGemBin + " " + strings.Join(config.passthroughOpts, " ") + " " + config.accountKey)
	cmd := exec.Command(config.scoutGemBin, cmdOpts...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent finished")
	agentRunning.Unlock()
	fmt.Println("agent available")
}

func loadConfig(cfg *Config) {
	var file = flag.String("file", "/etc/scout/scoutd.yml", "Configuration file in YAML format")

	flag.Parse()

	conf, err := yaml.ReadFile(*file)
	if err != nil {
		log.Fatalf("readfile(%q): %s\n", *file, err)
	}

	cfg.accountKey, err = conf.Get("account_key")
	if err != nil {
		log.Fatalf("Missing account_key")
	}

	cfg.gemBinPath, err = conf.Get("scout_gem_bin_path")
	if len(cfg.gemBinPath) == 0 {
		cfg.gemBinPath = "/usr/share/scout/gems/bin"
	}

	cfg.scoutGemBin = cfg.gemBinPath + "/scout"

	cfg.hostName, err = conf.Get("hostname")
	if len(cfg.hostName) == 0 {
		var hostname, err = os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
		cfg.hostName = strings.Split(hostname, ".")[0]
	}

	cfg.reportingServerUrl, err = conf.Get("reporting_server_url")
	if len(cfg.reportingServerUrl) != 0 {
		cfg.passthroughOpts = append(cfg.passthroughOpts, "-s", cfg.reportingServerUrl)
	}

	cfg.agentDataFile, err = conf.Get("agent_data_file")
	if len(cfg.agentDataFile) != 0 {
		cfg.passthroughOpts = append(cfg.passthroughOpts, "-d", cfg.agentDataFile)
	}
}
