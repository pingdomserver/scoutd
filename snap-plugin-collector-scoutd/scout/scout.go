package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"log"
	"time"
	"./statsd"
)

type scoutCollector struct {
	statsd interface{}
}

var config ScoutConfig
var activeCollectors map[string]statsd.Collector

func NewScoutCollector() *scoutCollector {
	sd := initStatsdCollector()

	return &scoutCollector { statsd: sd }
}

func initStatsdCollector() (*statsd.StatsdCollector) {
	activeCollectors = make(map[string]statsd.Collector)

	flushInterval := time.Duration(60) * time.Second
	if sd, err := statsd.NewStatsdCollector("statsd", "127.0.0.1:8125", flushInterval, 10); err != nil {
		log.Printf("error creating statsd collector: %s", err)
	} else {
		activeCollectors[sd.Name()] = sd

		sd.Start()

		return sd
	}
	return nil
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
	var ret []plugin.Metric

	payloads := make([]*statsd.CollectorPayload, len(activeCollectors))

	i := 0
	for _, c := range activeCollectors {
		pld := c.Payload()
		payloads[i] = pld
		i++
	}
	p := make(map[string][]*statsd.CollectorPayload, 1)
	p["collectors"] = payloads

	scoutClientMetrics, _ := RunScout()
	// convertedMetric := sc.ConvertMetric(scoutClientMetrics, "client");

	ret = append(ret, plugin.Metric{
		Data: scoutClientMetrics,
	})

	return ret, nil
}

func (sc *scoutCollector) ConvertMetric(m []byte, inputName string) []plugin.Metric {
	return []plugin.Metric {{ Data: m }}
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
