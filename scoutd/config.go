package scoutd

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/imdario/mergo"
)

type ScoutConfig struct {
	ConfigFile string
	AccountKey string
	HostName string
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
	PassthroughOpts []string
	SubCommand string
	HttpClients struct {
		HttpClient *http.Client
		HttpsClient *http.Client
	}
}

func LoadConfig(cfg *ScoutConfig) {
	var configFile string
	defaults := LoadDefaults()
	envOpts := LoadEnvOpts() // load the environment variables
	cliOpts := ParseOptions() // load the command line flags
	if cliOpts.ConfigFile != "" {
		configFile = cliOpts.ConfigFile
	} else if envOpts.ConfigFile != "" {
		configFile = envOpts.ConfigFile
	} else {
		configFile = defaults.ConfigFile
	}
	ymlOpts := LoadConfigFile(configFile) // load the options set in the config file
	//log.Printf("Defaults: %#v\n", defaults)
	//log.Printf("envOpts: %#v\n", envOpts)
	//log.Printf("cliOpts: %#v\n", cliOpts)
	//log.Printf("ymlOts: %#v\n", ymlOpts)
	if err := mergo.Merge(cfg, defaults); err != nil {
		log.Fatalf("Error while merging default config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, envOpts); err != nil {
		log.Fatalf("Error while merging environment config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, ymlOpts); err != nil {
		log.Fatalf("Error while merging YAML file config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, cliOpts); err != nil {
		log.Fatalf("Error while merging CLI config options: %s\n", err)
	}

	// Compile the passthroughOpts the scout ruby gem agent will need
	if cfg.AgentEnv != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-e", cfg.AgentEnv)
	}
	if cfg.AgentRoles != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-r", cfg.AgentRoles)
	}
	if cfg.ReportingServerUrl != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-s", cfg.ReportingServerUrl)
	}
	if cfg.AgentDataFile != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-d", cfg.AgentDataFile)
	}

	//log.Printf("Effective configuration: %#v\n", cfg)
	LoadHttpClients(cfg)
}

func LoadDefaults() (cfg ScoutConfig) {
	cfg.ConfigFile = "/etc/scout/scoutd.yml"
	cfg.HostName = ShortHostname()
	cfg.LogDir = "/var/log/scoutd"
	cfg.GemPath = "/usr/share/scout/gems"
	cfg.GemBinPath = cfg.GemPath + "/bin" 
	cfg.AgentGemBin = cfg.GemBinPath + "/scout"
	return
}

func LoadEnvOpts() (cfg ScoutConfig) {
	cfg.ConfigFile = os.Getenv("SCOUT_CONFIG_FILE")
	cfg.AccountKey = os.Getenv("SCOUT_ACCOUNT_KEY")
	cfg.HostName = os.Getenv("SCOUT_HOSTNAME")
	cfg.RunDir = os.Getenv("SCOUT_RUN_DIR")
	cfg.LogDir = os.Getenv("SCOUT_LOG_DIR")
	cfg.GemPath = os.Getenv("SCOUT_GEM_PATH")
	cfg.GemBinPath = os.Getenv("SCOUT_GEM_BIN_PATH")
	cfg.AgentGemBin = os.Getenv("SCOUT_AGENT_GEM_BIN")
	cfg.AgentEnv = os.Getenv("SCOUT_ENVIRONMENT")
	cfg.AgentRoles = os.Getenv("SCOUT_ROLES")
	cfg.AgentDataFile = os.Getenv("SCOUT_AGENT_DATA_FILE")
	cfg.HttpProxyUrl = os.Getenv("SCOUT_HTTP_PROXY")
	cfg.HttpsProxyUrl = os.Getenv("SCOUT_HTTPS_PROXY")
	if cfg.HttpProxyUrl == "" {
		cfg.HttpProxyUrl = os.Getenv("http_proxy")
	}
	if cfg.HttpsProxyUrl == "" {
		cfg.HttpsProxyUrl = os.Getenv("https_proxy")
	}
	cfg.ReportingServerUrl =os.Getenv("SCOUT_REPORTING_SERVER_URL")
	return
}

func LoadConfigFile(configFile string) (cfg ScoutConfig) {
	conf, err := yaml.ReadFile(configFile)
	if err != nil {
		log.Printf("Could not open config file: readfile(%q): %s\n", configFile, err)
		return
	}
	cfg.AccountKey, err = conf.Get("account_key")
	cfg.HostName, err = conf.Get("hostname")
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

func LoadHttpClients(cfg *ScoutConfig) {
	var secTr, plainTr *http.Transport
	var secProxyUrl, plainProxyUrl *url.URL
	var err error

	// set up the secure proxy and transport
	if cfg.HttpsProxyUrl != "" {
		secProxyUrl, err = url.Parse(cfg.HttpsProxyUrl)
		if err != nil {
			log.Fatalf("Error parsing HttpsProxyUrl: %s", err)
		}
	}
	secTr = &http.Transport{
		Proxy: http.ProxyURL(secProxyUrl),
	}
	cfg.HttpClients.HttpsClient = &http.Client{Transport: secTr}

	// Set up the plain proxy and transport
	if cfg.HttpProxyUrl != "" {
		plainProxyUrl, err = url.Parse(cfg.HttpProxyUrl)
		if err != nil {
			log.Fatalf("Error parsing HttpProxyUrl: %s", err)
		}
	}
	plainTr = &http.Transport{
		Proxy: http.ProxyURL(plainProxyUrl),
	}
	cfg.HttpClients.HttpClient = &http.Client{Transport: plainTr}
}