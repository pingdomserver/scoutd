package scout

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

var mTypes = []plugin.Metric{}
var collector = NewScoutCollector()

func TestShouldStartStatsdCollector(t *testing.T) {
	assert.NotNil(t, collector.statsd)
}

func TestShouldCollectMetrics(t *testing.T) {
	metrics, _ := collectMetrics()
	log.Printf("\n\nMETRICS: %s\n\n", metrics)
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

	return metrics, err
}
