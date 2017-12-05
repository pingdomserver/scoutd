package scout

import (
  "testing"
  "log"
  "github.com/stretchr/testify/assert"
  "github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
)

var mTypes = []plugin.Metric{}
var collector = NewScoutCollector()

func TestShouldStartStatsdCollector(t *testing.T) {
  assert.NotNil(t, collector.statsd)
}

func TestShouldCollectMetrics(t *testing.T) {
  metrics, _ := collectMetrics()
  log.Printf("metrics: %s", metrics)
  assert.NotNil(t, metrics)
}

func TestShouldCollectRubyClientMetrics(t *testing.T) {

}

func TestShouldCillectStatsdMetrics(t *testing.T) {

}

func TestSomethingElse(t *testing.T) {
  assert.Nil(t, nil)
}


func collectMetrics() ([]plugin.Metric, error) {
  metrics, err := collector.CollectMetrics(mTypes)
  log.Printf("KOLLEKT: %s", metrics)
  return metrics, err
}
