package scoutd

type DebugOptions struct {
	Outfile   string `short:"o" long:"outfile" description:"Write debug information to FILE" value-name:"FILE"`
	AssumeYes bool   `short:"y" description:"Overwrite FILE without asking. If this option is specified and FILE exists, you will be asked if you want to overwrite FILE."`
}

func init() {
	var debugOpts DebugOptions
	parser.AddCommand("debug", "Run a debug routine to check for any problems", "", &debugOpts)
}
