package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"log"
	"time"
	"os"
	// "bytes"
  // "bufio"
  "fmt"
	"./statsd"
	"encoding/json"
)

type scoutCollector struct {
	statsd interface{}
}

var activeCollectors map[string]statsd.Collector

func NewScoutCollector() *scoutCollector {
	sd := initStatsdCollector()

	return &scoutCollector { statsd: sd }
}

func initStatsdCollector() (*statsd.StatsdCollector) {
	log.Printf("gryby: init statsd")
	activeCollectors = make(map[string]statsd.Collector)

	flushInterval := time.Duration(60) * time.Second
	if sd, err := statsd.NewStatsdCollector("statsd", "127.0.0.1:8125", flushInterval, 10); err != nil {
		log.Printf("error creating statsd collector: %s", err)
	} else {
		activeCollectors[sd.Name()] = sd

		sd.Start()

		SavePayload([]byte("started statsd collector"))

		log.Printf("buraczki")
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
	SavePayload([]byte("CollectMetrics"))
	payloads := make([]*statsd.CollectorPayload, len(activeCollectors))

	i := 0
	for _, c := range activeCollectors {
		pld := c.Payload()
		payloads[i] = pld
		kiszka := fmt.Sprintf("PEJLOAD: %s", pld)
		SavePayload([]byte(kiszka))
		i++
	}
	p := make(map[string][]*statsd.CollectorPayload, 1)
	p["collectors"] = payloads

	js, err := json.Marshal(p)
	if err != nil {
		log.Printf("%s", err)
		SavePayload([]byte("KARTOFEL :C"))
	} else {
		SavePayload(js)
		log.Printf("majoenz: %s", js)
	}

	return nil, RunScout()
}

func SavePayload(payload []byte) {
  f, _ := os.OpenFile("/tmp/scout", os.O_APPEND|os.O_WRONLY, 0600)

  s := string(payload[:])

	log.Printf("KASZKA %s", s)
	// s := string(payload[n])
	f.WriteString(s)
	f.WriteString("\n")

  // w.Flush()
	f.Close()
}

func (scoutCollector) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	return *plugin.NewConfigPolicy(), nil
}
