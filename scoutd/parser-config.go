package scoutd

type GenConfigOptions struct {
	Outfile string `short:"o" long:"outfile" description:"Write generated configuration to FILE" value-name:"FILE"`
	AssumeYes bool `short:"y" description:"Overwrite FILE without asking. If this option is specified and FILE exists, you will be asked if you want to overwrite FILE."`
}

var genCfgOptions GenConfigOptions

func init() {
	parser.AddCommand("config", "Generate a config file based on the Application Options provided", "", &genCfgOptions)
}