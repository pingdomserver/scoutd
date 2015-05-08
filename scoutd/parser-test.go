package scoutd

type TestConfigOptions struct {
	// no options for the start command
}

func init() {
	var testCfgOptions TestConfigOptions
	parser.AddCommand("test", "Test a custom plugin", "", &testCfgOptions)
}
