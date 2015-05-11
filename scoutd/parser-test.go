package scoutd

type TestConfigOptions struct {
	TestArgs struct {
		PluginOptions []string `description:"Options to passed through to the plugin"`
	} `positional-args:"yes" required:"yes"`
}

var testCfgOptions TestConfigOptions

func init() {
	parser.AddCommand("test", "Test a custom plugin", "", &testCfgOptions)
}
