package scoutd

import (
	"log"
	"os"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/imdario/mergo"
)

type ScoutConfig struct {
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
	PassthroughOpts []string
	SubCommand string
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
	log.Printf("Defaults: %#v\n", defaults)
	log.Printf("envOpts: %#v\n", envOpts)
	log.Printf("cliOpts: %#v\n", cliOpts)
	log.Printf("ymlOts: %#v\n", ymlOpts)
	if err := mergo.Merge(cfg, defaults); err != nil {
		log.Fatalf("Error while merging default config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, envOpts); err != nil {
		log.Fatalf("Error while merging environment config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, cliOpts); err != nil {
		log.Fatalf("Error while merging CLI config options: %s\n", err)
	}
	if err := mergo.Merge(cfg, ymlOpts); err != nil {
		log.Fatalf("Error while merging YAML file config options: %s\n", err)
	}

	// Compile the passthroughOpts the scout ruby gem agent will need
	if cfg.ReportingServerUrl != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-s", cfg.ReportingServerUrl)
	}
	if cfg.AgentDataFile != "" {
		cfg.PassthroughOpts = append(cfg.PassthroughOpts, "-d", cfg.AgentDataFile)
	}

	log.Printf("Effective configuration: %#v\n", cfg)
}

func LoadDefaults() (cfg ScoutConfig) {
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

func LoadEnvOpts() (cfg ScoutConfig) {
	cfg.ConfigFile = os.Getenv("SCOUT_CONFIG_FILE")
	cfg.AccountKey = os.Getenv("SCOUT_ACCOUNT_KEY")
	cfg.HostName = os.Getenv("SCOUT_HOSTNAME")
	cfg.UserName = os.Getenv("SCOUT_USER")
	cfg.GroupName = os.Getenv("SCOUT_GROUP")
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