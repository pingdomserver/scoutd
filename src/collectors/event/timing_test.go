package event

import "testing"

//  Example Statsd percentile calculation of a timer
//  timers: { mytimer: [ 0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95 ] },
//  timer_data:
//   { mytimer:
//      { mean_90: 42.5,
//        upper_90: 85,
//        sum_90: 765,
//        std: 28.83140648667699,
//        upper: 95,
//        lower: 0,
//        count: 20,
//        count_ps: 2,
//        sum: 950,
//        mean: 47.5,
//        median: 47.5 } }

func TestStatsdCompatiblePercentile(t *testing.T) {
	e := NewTiming("statsd_compatible", 0)
	timerVals := []float64{5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}
	for _, v := range timerVals {
		e2 := NewTiming("new", v)
		e.Update(e2)
	}
	testStatsdPercentile20(t, e)
}

func TestPercentileOneValue(t *testing.T) {
	e := NewTiming("one_timer", 1)
	ps := e.Percentile(float64(0.9))
	if 1 != ps.mean {
		t.Errorf("90th percentile mean: 1 != %v\n", ps.mean)
	}
	if 1 != ps.upper {
		t.Errorf("90th percentile upper: 1 != %v\n", ps.upper)
	}
	if 1 != ps.sum {
		t.Errorf("90th percentile sum: 1 != %v\n", ps.sum)
	}
}

func TestPercentileAfterReset(t *testing.T) {
	e := NewTiming("reset_timer", 1)
	e.Reset()
	ps := e.Percentile(float64(0.9))
	if 0 != ps.mean {
		t.Errorf("90th percentile mean: 0 != %v\n", ps.mean)
	}
	if 0 != ps.upper {
		t.Errorf("90th percentile upper: 0 != %v\n", ps.upper)
	}
	if 0 != ps.sum {
		t.Errorf("90th percentile sum: 0 != %v\n", ps.sum)
	}
}

func TestUpdateAfterReset() {
	e := NewTiming("reset_timer", 1)
	e.Reset()
	for _, v := range []float64{5, 10} {
		e2 := NewTiming("new", v)
		e.Update(e2)
	}
	if min := e.Min; 5 != min {
		t.Errorf("Min: 5 != %v\n", min)
	}
	if max := e.Max; 10 != max {
		t.Errorf("Max: 10 != %v\n", max)
	}
	if value := e.Value; 5 != value {
		t.Errorf("Value: 15 != %v\n", value)
	}
	if count := e.Count; 2 != count {
		t.Errorf("Count: 2 != %v\n", count)
	}
}

func testStatsdPercentile20(t *testing.T, e *Timing) {
	if count := e.Count; 20 != count {
		t.Errorf("e.Count: 20 != %v\n", count)
	}
	if min := e.Min; 0 != min {
		t.Errorf("e.Min: 0 != %v\n", min)
	}
	if max := e.Max; 95 != max {
		t.Errorf("e.Max: 95 != %v\n", max)
	}
	ps := e.Percentile(float64(0.9))
	if 42.5 != ps.mean {
		t.Errorf("90th percentile mean: 42.5 != %v\n", ps.mean)
	}
	if 85 != ps.upper {
		t.Errorf("90th percentile upper: 85 != %v\n", ps.upper)
	}
	if 765 != ps.sum {
		t.Errorf("90th percentile sum: 765 != %v\n", ps.sum)
	}
	metrics := e.PercentileMetrics(0.9)
	sumName := "statsd_compatible.sum_90"
	if metrics[0].Name != sumName {
		t.Errorf("Percentile Metric Name %s != %v\n", sumName, metrics[0].Name)
	}
	if metrics[0].Value != 765 {
		t.Errorf("Percentile Metric Sum 765 != %v\n", metrics[0].Value)
	}
	meanName := "statsd_compatible.mean_90"
	if metrics[1].Name != meanName {
		t.Errorf("Percentile Metric Name %s != %v\n", meanName, metrics[1].Name)
	}
	if metrics[1].Value != 42.5 {
		t.Errorf("Percentile Metric Sum 42.5 != %v\n", metrics[1].Value)
	}
	upperName := "statsd_compatible.upper_90"
	if metrics[2].Name != upperName {
		t.Errorf("Percentile Metric Name %s != %v\n", upperName, metrics[2].Name)
	}
	if metrics[2].Value != 85 {
		t.Errorf("Percentile Metric Sum 85 != %v\n", metrics[2].Value)
	}
}