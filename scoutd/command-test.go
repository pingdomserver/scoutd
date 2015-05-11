package scoutd

import (
	"os/exec"
)

func RunTest(cfg ScoutConfig) {
	var out []byte
	var err error
	cmdOpts := append([]string{cfg.AgentRubyBin, "test"}, testCfgOptions.TestArgs.PluginOptions...)
	cmd := exec.Command(cfg.RubyPath, cmdOpts...)
	if out, err = cmd.CombinedOutput(); err != nil {
		cfg.Log.Printf("Error running agent: %s", err)
	}
	cfg.Log.Printf("Agent output:\n%s\n", out)
}