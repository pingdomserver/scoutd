package event

// constant event type identifiers
const (
	EventIncr = iota
	EventTiming
	EventAbsolute
	EventTotal
	EventGauge
	EventGaugeDelta
	EventFGauge
	EventFGaugeDelta
	EventFAbsolute
	EventPrecisionTiming
)

// A struct representing the metric in the json checkin bundle
type Metric struct {
	Name  string   `json:"name"`
	Value float64  `json:"value"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
}

// Event is an interface to a generic StatsD event
type Event interface {
	Metrics() []*Metric
	Type() int
	TypeString() string
	Payload() interface{}
	Update(e2 Event) error
	String() string
	Key() string
	SetKey(string)
}
