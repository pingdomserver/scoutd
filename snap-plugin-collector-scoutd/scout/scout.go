package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"log"
	"time"
	"./statsd"
	"encoding/json"
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

	if scoutClientMetrics, err := RunScout(); err != nil {
		log.Printf("error collection scout client metrics collector: %s", err)
	} else {
		// children, e := scoutClientMetrics.S("object").ChildrenMap()
		log.Printf("\n\n\tKLESZCZE: %s", scoutClientMetrics)
		parseClientMetrics(scoutClientMetrics)
	}
	return ret, nil
}

func parseClientMetrics(scoutClientMetrics []byte) map[string]interface{} {
	var checkinDataMap map[string]interface{}

	if err := json.Unmarshal(scoutClientMetrics, &checkinDataMap); err != nil {
		panic(err)
	}
	return parseClientMetricsMap(checkinDataMap)
}

func parseClientMetricsMap(checkinDataMap map[string]interface{}) map[string]interface{} {
	for key, child := range checkinDataMap {
		if rec, ok := child.(map[string]interface{}); ok {
			parseClientMetricsMap(rec)
		} else {
			log.Printf("key: %v, value: %v\n", key, child)
		}
	}
	return checkinDataMap
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
