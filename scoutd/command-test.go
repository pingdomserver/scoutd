package scoutd

import (
	"fmt"
	"os/exec"
)

func RunTest(cfg ScoutConfig) {
	var out []byte
	var err error
	cmdOpts := append([]string{cfg.AgentRubyBin, "test"}, testCfgOptions.TestArgs.PluginOptions...)
	cmd := exec.Command(cfg.RubyPath, cmdOpts...)
	if out, err = cmd.CombinedOutput(); err != nil {
		fmt.Printf("Error running agent: %s\n", err)
	}
	fmt.Printf("%s\n", out)
}