package event

import (
	"fmt"
	"sort"
)

type float64Slice []float64

func (p float64Slice) Sum() float64 {
	sum := 0.0
	for _, v := range p {
		sum = sum + v
	}
	return sum
}

func (p float64Slice) Mean() float64 {
	if p.Len() == 0 {
		return 0
	}
	return p.Sum() / float64(p.Len())
}

func (p float64Slice) Len() int           { return len(p) }
func (p float64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p float64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p float64Slice) PercentileSummary(pct float64) *PercentileSummary {
	ps := &PercentileSummary{}
	ps.threshold = pct
	ps.thresholdString = fmt.Sprintf("%.0f", pct * 100)
	count := len(p)
	if (count > 1) {
		sort.Sort(p)
		nrThreshold := int((pct * float64(count)) + 0.5)
		threshSlice := p[:nrThreshold]
		ps.sum = threshSlice.Sum()
		ps.mean = ps.sum / float64(len(threshSlice))
		ps.upper = threshSlice[len(threshSlice) - 1]
	} else if (count > 0) {
		ps.sum = p[0]
		ps.mean = p[0]
		ps.upper = p[0]
	}
	return ps
}

type PercentileSummary struct {
	threshold float64
	thresholdString string
	mean   float64
	sum    float64
	upper  float64
}

// Timing keeps min/max/mean information about a timer over a certain interval
type Timing struct {
	Name  string
	Min   float64
	Max   float64
	Value float64
	Values float64Slice
	Count float64
	Tags  []string
}

// NewTiming is a factory for a Timing event, setting the Count to 1 to prevent div_by_0 errors
func NewTiming(k string, delta float64) *Timing {
	fs := []float64{delta}
	return &Timing{Name: k, Min: delta, Max: delta, Value: delta, Values: float64Slice(fs), Count: 1, Tags: []string{}}
}

func (e *Timing) Percentile(pct float64) *PercentileSummary {
	return e.Values.PercentileSummary(pct)
}

func (e *Timing) PercentileMetrics(pct float64) []*Metric {
	ps := e.Percentile(pct)
	return []*Metric{
		{fmt.Sprintf("%s.sum_%s", e.Name, ps.thresholdString), ps.sum, "timer", e.Tags},
		{fmt.Sprintf("%s.mean_%s", e.Name, ps.thresholdString), ps.mean, "timer", e.Tags},
		{fmt.Sprintf("%s.upper_%s", e.Name, ps.thresholdString), ps.upper, "timer", e.Tags},
	}
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Timing) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	p := e2.Payload().(map[string]float64)
	e.Count += p["cnt"]
	e.Value += p["val"]
	e.Values = append(e.Values, p["val"])
	e.Min = minFloat64(e.Min, p["min"])
	e.Max = maxFloat64(e.Max, p["max"])
	e.Tags = []string{}
	return nil
}

// Resets Min/Max/Value/Count to 0
func (e *Timing) Reset() {
	e.Min = 0
	e.Max = 0
	e.Value = 0
	e.Values = make(float64Slice, 0)
	e.Count = 0
}

// Return a copy of this Timing event
func (e *Timing) Copy() Event {
	e2 := &Timing{Name: e.Name, Min: e.Min, Max: e.Max, Value: e.Value, Values: e.Values, Count: e.Count, Tags: e.Tags}
	return e2
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
	var meanVal float64
	if e.Count > 0 {
		meanVal = float64(e.Value / e.Count) // make sure e.Count != 0
	}
	pctMetrics := e.PercentileMetrics(0.95)
	metrics := []*Metric{
		{fmt.Sprintf("%s.count", e.Name), e.Count, "timer", e.Tags},
		{fmt.Sprintf("%s.sum", e.Name), e.Value, "timer", e.Tags},
		{fmt.Sprintf("%s.mean", e.Name), meanVal, "timer", e.Tags},
		{fmt.Sprintf("%s.min", e.Name), e.Min, "timer", e.Tags},
		{fmt.Sprintf("%s.max", e.Name), e.Max, "timer", e.Tags},
	}
	metrics = append(metrics, pctMetrics...)
	return metrics
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
