package scout

import (
	"encoding/json"
	"log"
	"time"
	"fmt"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"github.com/pingdomserver/scoutd/collectors"
	"github.com/pingdomserver/scoutd/scoutd"
	"github.com/buger/jsonparser"
)

type scoutCollector struct {
	statsd interface{}

	scoutClient []plugin.Metric
}

var config scoutd.ScoutConfig
var activeCollectors map[string]collectors.Collector

const baseMetric string = "/solarwinds/psm/metrics"

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

		// sc.parseClientMetrics(scoutClientMetrics)
		sc.parseMetrics("server_metrics", baseMetric, scoutClientMetrics)
		// sc.parsePluginMetrics(scoutClientMetrics)
	}
	return sc.scoutClient, nil
}

// func (sc *scoutCollector) parseClientMetrics(scoutClientMetrics []byte) map[string]interface{} {
// 	var checkinDataMap map[string]interface{}
//
// 	if err := json.Unmarshal(scoutClientMetrics, &checkinDataMap); err != nil {
// 		panic(err)
// 	}
// 	log.Printf("\n\nBonCYZSLAW: %s", checkinDataMap)
//
// 	return sc.parseClientMetricsMap(baseMetricNamespace, checkinDataMap)
// }

// Parse client metrics
func (sc *scoutCollector) parseMetrics(namespace string, metric string, scoutClientMetrics []byte) {
	log.Printf("NEJMSPAJES: %s, METRIKS: %s", namespace, scoutClientMetrics)
	jsonparser.ObjectEach(scoutClientMetrics, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if (dataType == jsonparser.Object) {
			keys := sc.getJsonKeys(value)
			for k := range keys {
				metricName := fmt.Sprintf("%s/%s", metric, keys[k])
				sc.parseMetrics(keys[k], metricName, value)
			}
		} else {
			name := fmt.Sprintf("%s/%s", metric, key)
			log.Printf("metricName: %s --- metric value: ", name, string(value))
			log.Printf("BOROWIKI: %s PIECZARKI: %s", key, value)
			metrics := append(sc.scoutClient, plugin.Metric{
				Namespace: plugin.NewNamespace(name),
				Data:      string(value),
			})
			sc.scoutClient = metrics
		}
		return nil
	}, namespace)
}

func (sc *scoutCollector) getJsonKeys(data []byte) []string {
	c := make(map[string]interface{})
	e := json.Unmarshal(data, &c)
	if e != nil {
		panic(e)
	}
	k := make([]string, len(c))
		// iteration counter
	i := 0

	// copy c's keys into k
	for s, _ := range c {
			k[i] = s
			i++
	}
	return k
}

func (sc *scoutCollector) parsePluginMetrics(scoutClientMetrics []byte) {
	log.Printf("PARS Plugin METRICS")
	jsonparser.ArrayEach(scoutClientMetrics, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		fmt.Println(jsonparser.Get(value, "plugin_id"))
	}, "reports")
}

// func (sc *scoutCollector) parseClientMetricsMap(mapKey string, checkinDataMap map[string]interface{}) map[string]interface{} {
// 	for key, child := range checkinDataMap {
// 		newKey := mapKey
// 		if key != "" {
// 			newKey = mapKey + "/" + key
// 		}
// 		if rec, ok := child.(map[string]interface{}); ok {
// 			sc.parseClientMetricsMap(newKey, rec)
// 		} else {
// 			log.Printf("key: %v, value: %v\n", newKey, child)
//
// 			majonez := append(sc.scoutClient, plugin.Metric{
// 				Namespace: plugin.NewNamespace(newKey),
// 				Data:      child,
// 			})
// 			sc.scoutClient = majonez
// 		}
// 	}
// 	return checkinDataMap
// }

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
