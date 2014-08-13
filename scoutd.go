package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oguzbilgic/pusher"
	"github.com/kylelemons/go-gypsy/yaml"
	"code.google.com/p/opts-go"
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
	agentGemBin string
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

	parseOptions(&config) // load the command line flags into global struct 'config'
	loadConfig(&config) // load the yaml configuration into global struct 'config'

	fmt.Printf("Config: %s\n", config)
	os.Exit(0)

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
		fmt.Printf(config.agentGemBin + " realtime " + msg.(string))
		cmd := exec.Command(config.agentGemBin, "realtime", msg.(string))
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

	fmt.Println("running agent: " + config.agentGemBin + " " + strings.Join(config.passthroughOpts, " ") + " " + config.accountKey)
	cmd := exec.Command(config.agentGemBin, cmdOpts...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent finished")
	agentRunning.Unlock()
	fmt.Println("agent available")
}

func loadConfig(cfg *Config) {
	conf, err := yaml.ReadFile(cfg.configFile)
	if err != nil {
		log.Fatalf("readfile(%q): %s\n", cfg.configFile, err)
	}

	cfg.accountKey, err = conf.Get("account_key")
	if err != nil {
		log.Fatalf("Missing account_key")
	}

	cfg.gemBinPath, err = conf.Get("scout_gem_bin_path")
	if len(cfg.gemBinPath) == 0 {
		cfg.gemBinPath = "/usr/share/scout/gems/bin"
	}

	cfg.agentGemBin = cfg.gemBinPath + "/scout"

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

func parseOptions(cfg *Config) {
	configFile := opts.Single("-f", "", "Configuration file to read, in YAML format", "")
	accountKey := opts.Single("-k", "--key", "Your account key", "")
	hostName := opts.Single("", "--hostname", "Report to the scout server as this hostname", "")
	userName := opts.Single("-u", "--user", "Run as this user", "")
	groupName := opts.Single("-g", "--group", "Run as this group", "")
	runDir := opts.Single("", "--rundir", "Set the working directory", "")
	logDir := opts.Single("", "--logdir", "Write logs to this directory", "")
	gemPath := opts.Single("", "--gempath", "Append this path to GEM_PATH before running the agent", "")
	gemBinPath := opts.Single("", "--gembinpath", "The path to the Gem binary directory", "")
	agentGemBin := opts.Single("", "--agentgembin", "The full path to the scout agent ruby gem", "")
	agentEnv := opts.Single("-e", "--environment", "Environment for this server. Environments are defined through scoutapp.com's web UI", "")
	agentRoles := opts.Single("-r", "--roles", "Roles for this server. Roles are defined through scoutapp.com's web UI", "")
	agentDataFile := opts.Single("-d", "--data", "The data file used to track history", "")
	httpProxyUrl := opts.Single("", "--http-proxy", "Optional http proxy for non-SSL traffic", "")
	httpsProxyUrl := opts.Single("", "--https-proxy", "Optional https proxy for SSL traffic.", "")
	reportingServerUrl := opts.Single("-s", "--server", "The URL for the server to report to.", "")

	opts.Parse()
	// There's probably an easier way to handle parsing these with reflection,
	// but for now I am just listing them explicitly to get things going - Dave
	if *configFile != "" {
		cfg.configFile = string(*configFile)
	}
	if *accountKey != "" {
		cfg.accountKey = string(*accountKey)
	}
	if *hostName != "" {
		cfg.hostName = string(*hostName)
	}
	if *userName != "" {
		cfg.userName = string(*userName)
	}
	if *groupName != "" {
		cfg.groupName = string(*groupName)
	}
	if *runDir != "" {
		cfg.runDir = string(*runDir)
	}
	if *logDir != "" {
		cfg.logDir = string(*logDir)
	}
	if *gemPath != "" {
		cfg.gemPath = string(*gemPath)
	}
	if *gemBinPath != "" {
		cfg.gemBinPath = string(*gemBinPath)
	}
	if *agentGemBin != "" {
		cfg.agentGemBin = string(*agentGemBin)
	}
	if *agentEnv != "" {
		cfg.agentEnv = string(*agentEnv)
	}
	if *agentRoles != "" {
		cfg.agentRoles = string(*agentRoles)
	}
	if *agentDataFile != "" {
		cfg.agentDataFile = string(*agentDataFile)
	}
	if *httpProxyUrl != "" {
		cfg.httpProxyUrl = string(*httpProxyUrl)
	}
	if *httpsProxyUrl != "" {
		cfg.httpsProxyUrl = string(*httpsProxyUrl)
	}
	if *reportingServerUrl != "" {
		cfg.reportingServerUrl = string(*reportingServerUrl)
	}
}
