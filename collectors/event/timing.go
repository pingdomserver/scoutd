package event

import "fmt"

// Timing keeps min/max/avg information about a timer over a certain interval
type Timing struct {
	Name  string
	Min   float64
	Max   float64
	Value float64
	Count float64
	Tags  []string
}

// NewTiming is a factory for a Timing event, setting the Count to 1 to prevent div_by_0 errors
func NewTiming(k string, delta float64) *Timing {
	return &Timing{Name: k, Min: delta, Max: delta, Value: delta, Count: 1, Tags: []string{}}
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Timing) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	p := e2.Payload().(map[string]float64)
	e.Count += p["cnt"]
	e.Value += p["val"]
	e.Min = minFloat64(e.Min, p["min"])
	e.Max = maxFloat64(e.Max, p["max"])
	return nil
}

// Payload returns the aggregated value for this event
func (e Timing) Payload() interface{} {
	return map[string]float64{
		"min": e.Min,
		"max": e.Max,
		"val": e.Value,
		"cnt": e.Count,
	}
}

func (e Timing) Metrics() []*Metric {
	if e.Count <= 0 {
		return []*Metric{}
	}
	return []*Metric{
		{fmt.Sprintf("%s.count", e.Name), e.Value, "counter", e.Tags},
		{fmt.Sprintf("%s.avg", e.Name), float64(e.Value / e.Count), "gauge", e.Tags}, // make sure e.Count != 0
		{fmt.Sprintf("%s.min", e.Name), e.Min, "gauge", e.Tags},
		{fmt.Sprintf("%s.max", e.Name), e.Max, "gauge", e.Tags},
	}
}

// Key returns the name of this metric
func (e Timing) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Timing) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Timing) Type() int {
	return EventTiming
}

// TypeString returns a name for this type of metric
func (e Timing) TypeString() string {
	return "Timing"
}

// String returns a debug-friendly representation of this metric
func (e Timing) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %+v}", e.TypeString(), e.Name, e.Payload())
}

func minFloat64(v1, v2 float64) float64 {
	if v1 <= v2 {
		return v1
	}
	return v2
}
func maxFloat64(v1, v2 float64) float64 {
	if v1 >= v2 {
		return v1
	}
	return v2
}
