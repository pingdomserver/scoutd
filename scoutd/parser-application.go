package scoutd

import (
	"os"
	flags "github.com/jessevdk/go-flags"
)

type ApplicationOptions struct {
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

var parser = flags.NewParser(nil, flags.Default)
var cliOpts ApplicationOptions

func init() {
	parser.AddGroup("Application Options", "", &cliOpts)
}

func ParseOptions() (cfg ScoutConfig) {
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

