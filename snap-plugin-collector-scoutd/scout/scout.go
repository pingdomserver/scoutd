package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"./statsd"
	"log"
)

type scoutCollector struct {
	dupa string
}

func NewScoutCollector() *scoutCollector {
	aa := statsd.Start()
	return &scoutCollector { dupa: aa }
}

func (scoutCollector) GetMetricTypes(config plugin.Config) ([]plugin.Metric, error) {
	return []plugin.Metric{getScoutMetricType()}, nil
}

func getScoutMetricType() plugin.Metric {
	return plugin.Metric{
		Namespace: plugin.NewNamespace("solarwinds", "scout", "metrics"),
	}
}

func (sc *scoutCollector) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	log.Printf("majoenz: %s", sc.dupa)
	return nil, RunScout()
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
