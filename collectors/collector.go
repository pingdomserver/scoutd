package collectors

import (
	"encoding/json"
	"github.com/scoutserver/scoutd/collectors/event"
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
	ReceiveCollectorMessage(CollectorMessage)
}

type CollectorMessage struct {
	SourceName string `json:"source_name"`
	SourceType string `json:"source_type"`
	MessageType string `json:"message_type"`
	Data        json.RawMessage `json:"data"`
}

// A struct representing the Collector's data in the json checkin bundle
type CollectorPayload struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Metrics []*event.Metric `json:"metrics"`
}
