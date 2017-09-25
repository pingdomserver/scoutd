package scout

import "github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"

type scoutCollector struct{}

func NewScoutCollector() scoutCollector {
	return scoutCollector{}
}

func (scoutCollector) GetMetricTypes(config plugin.Config) ([]plugin.Metric, error) {
	return []plugin.Metric{getScoutMetricType()}, nil
}

func getScoutMetricType() plugin.Metric {
	return plugin.Metric{
		Namespace: plugin.NewNamespace("solarwinds", "scout", "metrics"),
	}
}

func (scoutCollector) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	return nil, RunScout()
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
