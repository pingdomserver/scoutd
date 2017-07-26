package scout

import "github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"

// import "os/exec"

type scoutCollector struct{}

// Creates an instance of the scout/snap collector plugin.
func NewScoutCollector() plugin.Collector {
	return scoutCollector{}
}

func (scoutCollector) GetMetricTypes(config plugin.Config) ([]plugin.Metric, error) {
	metrics := []plugin.Metric{}
	return metrics, nil
}

func (scoutCollector) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	// TODO just fire scoutd and ruby part reporting directly to psm.

	error := RunScout()
	if error != nil {
		// TODO log or something
	}
	return nil, nil
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	return *policy, nil
}
