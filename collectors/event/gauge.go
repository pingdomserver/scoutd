package event

import "fmt"

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again
type Gauge struct {
	Name  string
	Value float64
	Tags  []string
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Gauge) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value += e2.Payload().(float64)
	e.Tags = []string{}
	return nil
}

// Payload returns the aggregated value for this event
func (e Gauge) Payload() interface{} {
	return e.Value
}

func (e Gauge) Metrics() []*Metric {
	return []*Metric{
		{e.Name, e.Value, "gauge", e.Tags},
	}
}

// Key returns the name of this metric
func (e Gauge) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Gauge) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Gauge) Type() int {
	return EventGauge
}

// TypeString returns a name for this type of metric
func (e Gauge) TypeString() string {
	return "Gauge"
}

// String returns a debug-friendly representation of this metric
func (e Gauge) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %f}", e.TypeString(), e.Name, e.Value)
}
