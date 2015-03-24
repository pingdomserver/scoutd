package event

import "fmt"

// Increment represents a metric whose value is averaged over a minute
type Increment struct {
	Name  string
	Value float64
	Tags  []string
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Increment) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value += e2.Payload().(float64)
	e.Tags = []string{}
	return nil
}

// Resets the Value to 0
func (e *Increment) Reset() {
	e.Value = 0
	return
}

func (e *Increment) Copy() Event {
	e2 := &Increment{Name: e.Name, Value: e.Value, Tags: e.Tags}
	return e2
}

// Payload returns the aggregated value for this event
func (e Increment) Payload() interface{} {
	return e.Value
}

// Stats returns an array of StatsD events as they travel over UDP
func (e Increment) Metrics() []*Metric {
	return []*Metric{
		{e.Name, e.Value, "counter", e.Tags},
	}
}

// Key returns the name of this metric
func (e Increment) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Increment) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Increment) Type() int {
	return EventIncr
}

// TypeString returns a name for this type of metric
func (e Increment) TypeString() string {
	return "Increment"
}

// String returns a debug-friendly representation of this metric
func (e Increment) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %f}", e.TypeString(), e.Name, e.Value)
}
