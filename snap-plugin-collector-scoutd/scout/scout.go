package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"./collectors"
	"time"
	"log"
)

var config ScoutConfig

type scoutCollector struct {
	Payload AgentPayload
}

func NewScoutCollector() scoutCollector {
	payload := AgentPayload {}
	flushInterval := time.Duration(60) * time.Second
	if statsd, err := collectors.NewStatsdCollector("statsd", config.Statsd.Addr, flushInterval, collectors.DefaultEventLimit); err != nil {
		config.Log.Printf("error creating statsd collector: %s", err)
	} else {
		statsd.Start()
	}

	return scoutCollector {
		Payload: payload,
	}
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
	log.Printf("kleszczu %s", plugin.Metric)
	return nil, RunScout()
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
