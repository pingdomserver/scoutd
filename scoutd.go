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
	flags "github.com/jessevdk/go-flags"
	"github.com/imdario/mergo"
	// "kylelemons.net/go/daemon"
)

type Config struct {
	ConfigFile string
	AccountKey string
	HostName string
	UserName string
	GroupName string
	RunDir string
	LogDir string
	GemPath string
	GemBinPath string
	AgentGemBin string
	AgentEnv string
	AgentRoles string
	AgentDataFile string
	HttpProxyUrl string
	HttpsProxyUrl string
	ReportingServerUrl string
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
	cmdOpts := append(config.passthroughOpts, config.AccountKey)

	fmt.Println("running agent: " + config.AgentGemBin + " " + strings.Join(config.passthroughOpts, " ") + " " + config.AccountKey)
	cmd := exec.Command(config.AgentGemBin, cmdOpts...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent finished")
	agentRunning.Unlock()
	fmt.Println("agent available")
}

func loadConfig(cfg *Config) {
	var configFile string
	defaults := loadDefaults()
	// envOpts = loadEnvOpts()
	cliOpts := parseOptions() // load the command line flags
	if cliOpts.ConfigFile != "" {
		configFile = cliOpts.ConfigFile
	} else {
		configFile = defaults.ConfigFile
	}
	ymlOpts := loadConfigFile(configFile) // load the options set in the config file
	fmt.Println("Defaults: ", defaults)
	fmt.Println("cliOpts: ", cliOpts)
	fmt.Println("ymlOts: ", ymlOpts)
	if err := mergo.Merge(&config, defaults); err != nil {
		log.Fatalf("Error while merging default config options: %s\n", err)
	}
	if err := mergo.Merge(&config, cliOpts); err != nil {
		log.Fatalf("Error while merging CLI config options: %s\n", err)
	}
	if err := mergo.Merge(&config, ymlOpts); err != nil {
		log.Fatalf("Error while merging YAML file config options: %s\n", err)
	}

	// Compile the passthroughOpts the scout ruby gem agent will need
	if config.ReportingServerUrl != "" {
		config.passthroughOpts = append(config.passthroughOpts, "-s", config.ReportingServerUrl)
	}
	if cfg.AgentDataFile != "" {
		config.passthroughOpts = append(config.passthroughOpts, "-d", config.AgentDataFile)
	}

	fmt.Println("Merged config: ", config)
}

func loadDefaults() (cfg Config) {
	cfg.ConfigFile = "/etc/scout/scoutd.yml"
	cfg.HostName = ShortHostname()
	cfg.UserName = "scoutd"
	cfg.GroupName = "scoutd"
	cfg.RunDir = "/var/run/scoutd"
	cfg.LogDir = "/var/log/scoutd"
	cfg.GemPath = "/usr/share/scout/gems"
	cfg.GemBinPath = cfg.GemPath + "/bin" 
	cfg.AgentGemBin = cfg.GemBinPath + "/scout"
	return
}

func loadConfigFile(configFile string) (cfg Config) {
	conf, err := yaml.ReadFile(configFile)
	if err != nil {
		log.Fatalf("readfile(%q): %s\n", configFile, err)
	}
	cfg.AccountKey, err = conf.Get("account_key")
	cfg.GemBinPath, err = conf.Get("gem_bin_path")
	cfg.AgentGemBin, err = conf.Get("agent_gem_bin")
	cfg.HostName, err = conf.Get("hostname")
	cfg.UserName, err = conf.Get("user")
	cfg.GroupName, err = conf.Get("group")
	cfg.RunDir, err = conf.Get("run_dir")
	cfg.LogDir, err = conf.Get("log_dir")
	cfg.GemPath, err = conf.Get("gem_path")
	cfg.GemBinPath, err = conf.Get("gem_bin_path")
	cfg.AgentGemBin, err = conf.Get("agent_gem_bin")
	cfg.AgentEnv, err = conf.Get("environment")
	cfg.AgentRoles, err = conf.Get("roles")
	cfg.AgentDataFile, err = conf.Get("agent_data_file")
	cfg.HttpProxyUrl, err = conf.Get("http_proxy")
	cfg.HttpsProxyUrl, err = conf.Get("https_proxy")
	cfg.ReportingServerUrl, err = conf.Get("reporting_server_url")
	return
}

func parseOptions() (cfg Config) {
	type CLIOptions struct {
		ConfigFile string `short:"f" long:"config" description:"Configuration file to read, in YAML format"`
		AccountKey string `short:"k" long:"key" description:"Your account key"`
		HostName string `long:"hostname" description:"Report to the scout server as this hostname"`
		UserName string `short:"u" long:"user" description:"Run as this user"`
		GroupName string `short:"g" long:"group" description:"Run as this group"`
		RunDir string `long:"rundir" description:"Set the working directory"`
		LogDir string `long:"logdir" description:"Write logs to this directory"`
		GemPath string `long:"gem_path" description:"Append this path to GEM_PATH before running the agent"`
		GemBinPath string `long:"gem-bin-path" description:"The path to the Gem binary directory"`
		AgentGemBin string `long:"agent-gem-bin" description:"The full path to the scout agent ruby gem"`
		AgentEnv string `short:"e" long:"environment" description:"Environment for this server. Environments are defined through scoutapp.com's web UI"`
		AgentRoles string `short:"r" long:"roles" description:"Roles for this server. Roles are defined through scoutapp.com's web UI"`
		AgentDataFile string `short:"d" long:"data" description:"The data file used to track history"`
		HttpProxyUrl string `long:"http-proxy" description:"Optional http proxy for non-SSL traffic"`
		HttpsProxyUrl string `long:"https-proxy" description:"Optional https proxy for SSL traffic."`
		ReportingServerUrl string `short:"s" long:"server" description:"The URL for the server to report to."`
	}
	var cliOpts CLIOptions
	parser := flags.NewParser(&cliOpts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
	cfg.ConfigFile = cliOpts.ConfigFile
	cfg.AccountKey = cliOpts.AccountKey
	cfg.HostName = cliOpts.HostName
	cfg.UserName = cliOpts.UserName
	cfg.GroupName = cliOpts.GroupName
	cfg.RunDir = cliOpts.RunDir
	cfg.LogDir = cliOpts.LogDir
	cfg.GemPath = cliOpts.GemPath
	cfg.GemBinPath = cliOpts.GemBinPath
	cfg.AgentGemBin = cliOpts.AgentGemBin
	cfg.AgentEnv = cliOpts.AgentEnv
	cfg.AgentRoles = cliOpts.AgentRoles
	cfg.AgentDataFile = cliOpts.AgentDataFile
	cfg.HttpProxyUrl = cliOpts.HttpProxyUrl
	cfg.HttpsProxyUrl = cliOpts.HttpsProxyUrl
	cfg.ReportingServerUrl = cliOpts.ReportingServerUrl
	return
}

func ShortHostname() string {
	var hostname, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(hostname, ".")[0]
}