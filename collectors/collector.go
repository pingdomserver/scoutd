package collectors

import (
	"github.com/scoutapp/scoutd/collectors/event"
)

const (
	StatsdType = iota
)

type Collector interface {
	Name() string
	Type() int
	TypeString() string
	Start()
	Collect() error
	Payload() *CollectorPayload
}

// A struct representing the Collector's data in the json checkin bundle
type CollectorPayload struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Metrics []*event.Metric `json:"metrics"`
}
