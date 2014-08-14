package scoutd

type StartConfigOptions struct {
	// no options for the start command
}

func init() {
	var startCfgOptions StartConfigOptions
	parser.AddCommand("start", "Start the scoutd daemon", "", &startCfgOptions)
}