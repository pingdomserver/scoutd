package scoutd

import "github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
import "scoutd"
import "os"

// import "os/exec"

type ScoutCollector struct{}

// Required comment.
func NewScoutCollector() ScoutCollector {
	return ScoutCollector{}
}

func (ScoutCollector) GetMetricTypes(config plugin.Config) ([]plugin.Metric, error) {
	metrics := []plugin.Metric{}
	return metrics, nil
}

func (ScoutCollector) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	// TODO just fire scoutd and ruby part reporting directly to psm.

	// os.Args[1]
	// main.StartScoutd()

	// const scoutdPath = "scoutd"
	// const scoutdArgs = "shot"
	// error := exec.Command(scoutdPath, scoutdArgs).Run()
	// if error != nil {
	// 	// TODO log error
	// }

	// TODO ugly as hell, but...

	os.Args = []string{os.Args[0], "start"} // "shot"}
	scoutd.StartScoutd()
	return nil, nil
}

func (ScoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	return *policy, nil
}
