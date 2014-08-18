package scoutd

type GenConfigOptions struct {
	AssumeYes bool `short:"y" description:"Overwrite FILE without asking. If this option is specified and FILE exists, you will be asked if you want to overwrite FILE."`
	Outfile string `short:"o" long:"outfile" description:"Write generated configuration to FILE" optional:"true" optional-value:"DEFAULT_VALUE" value-name:"FILE"`
}

var genCfgOptions GenConfigOptions

func init() {
	parser.AddCommand("config", "Generate a config file based on the Application Options provided", "", &genCfgOptions)
}