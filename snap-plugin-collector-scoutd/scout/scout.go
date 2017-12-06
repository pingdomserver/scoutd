package scout

import (
	"encoding/json"
	"log"
	"time"

	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"github.com/pingdomserver/scoutd/collectors"
	"github.com/pingdomserver/scoutd/scoutd"
)

type scoutCollector struct {
	statsd interface{}

	scoutClient []plugin.Metric
}

var config scoutd.ScoutConfig
var activeCollectors map[string]collectors.Collector

const baseMetricNamespace string = "/solarwinds/psm/metrics"

func NewScoutCollector() *scoutCollector {
	sd := initStatsdCollector()

	return &scoutCollector{statsd: sd}
}

func initStatsdCollector() *collectors.StatsdCollector {
	activeCollectors = make(map[string]collectors.Collector)

	flushInterval := time.Duration(60) * time.Second
	if sd, err := collectors.NewStatsdCollector("statsd", "127.0.0.1:8125", flushInterval, 10); err != nil {
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
		Namespace: plugin.NewNamespace("solarwinds", "psm", "metrics"),
	}
}

func (sc *scoutCollector) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	if scoutClientMetrics, err := RunScout(); err != nil {
		log.Printf("error collection scout client metrics collector: %s", err)
	} else {
		log.Printf("\n\n\tKLESZCZE: %s", scoutClientMetrics)

		sc.parseClientMetrics(scoutClientMetrics)
	}
	return sc.scoutClient, nil
}

func (sc *scoutCollector) parseClientMetrics(scoutClientMetrics []byte) map[string]interface{} {
	var checkinDataMap map[string]interface{}

	if err := json.Unmarshal(scoutClientMetrics, &checkinDataMap); err != nil {
		panic(err)
	}
	log.Printf("\n\nBonCYZSLAW: %s", checkinDataMap)

	return sc.parseClientMetricsMap(baseMetricNamespace, checkinDataMap)
}

func (sc *scoutCollector) parseClientMetricsMap(mapKey string, checkinDataMap map[string]interface{}) map[string]interface{} {
	for key, child := range checkinDataMap {
		newKey := mapKey
		if key != "" {
			newKey = mapKey + "/" + key
		}
		if rec, ok := child.(map[string]interface{}); ok {
			sc.parseClientMetricsMap(newKey, rec)
		} else {
			log.Printf("key: %v, value: %v\n", newKey, child)

			majonez := append(sc.scoutClient, plugin.Metric{
				Namespace: plugin.NewNamespace(newKey),
				Data:      child,
			})
			sc.scoutClient = majonez
		}
	}
	return checkinDataMap
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
