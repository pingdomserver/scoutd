package scoutd

import (
	flags "github.com/scoutapp/go-flags"
	"os"
)

type ApplicationOptions struct {
	ConfigFile         string `short:"f" long:"config" description:"Configuration file to read, in YAML format"`
	AccountKey         string `short:"k" long:"key" description:"Your account key"`
	HostName           string `long:"hostname" description:"Report to the scout server as this hostname"`
	RunDir             string `long:"rundir" description:"Set the working directory"`
	LogFile            string `long:"logfile" description:"Write logs to FILE. Write to STDOUT if FILE is '-'"`
	RubyPath           string `long:"ruby-path" description:"The full path to the ruby binary used to run the scout ruby client"`
	AgentRubyBin       string `long:"agent-ruby-bin" description:"The full path to the scout ruby agent"`
	AgentEnv           string `short:"e" long:"environment" description:"Environment for this server. Environments are defined through scoutapp.com's web UI"`
	AgentRoles         string `short:"r" long:"roles" description:"Roles for this server. Roles are defined through scoutapp.com's web UI"`
	AgentDisplayName   string `short:"n" long:"name" description:"Optional name to display for this server on scoutapp.com's web UI"`
	AgentDataFile      string `short:"d" long:"data" description:"The data file used to track history"`
	HttpProxyUrl       string `long:"http-proxy" description:"Optional http proxy for non-SSL traffic"`
	HttpsProxyUrl      string `long:"https-proxy" description:"Optional https proxy for SSL traffic."`
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
	cfg.RunDir = cliOpts.RunDir
	cfg.LogFile = cliOpts.LogFile
	cfg.RubyPath = cliOpts.RubyPath
	cfg.AgentRubyBin = cliOpts.AgentRubyBin
	cfg.AgentEnv = cliOpts.AgentEnv
	cfg.AgentRoles = cliOpts.AgentRoles
	cfg.AgentDisplayName = cliOpts.AgentDisplayName
	cfg.AgentDataFile = cliOpts.AgentDataFile
	cfg.HttpProxyUrl = cliOpts.HttpProxyUrl
	cfg.HttpsProxyUrl = cliOpts.HttpsProxyUrl
	cfg.ReportingServerUrl = cliOpts.ReportingServerUrl
	cfg.SubCommand = parser.Command.Active.Name
	return
}
