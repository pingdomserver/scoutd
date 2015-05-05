package collectors

import "testing"

func TestStatsdParseLine(t *testing.T) {
	var line []byte
	var err error

	line = []byte(":")
	_, err = parseLine(line)
	if err == nil {
		t.Errorf("No error on no name or value: %s", line)
	}

	line = []byte("namenovalue:|c")
	_, err = parseLine(line)
	if err == nil {
		t.Errorf("No error on no value: %s", line)
	}

	line = []byte(":1.05|c")
	_, err = parseLine(line)
	if err == nil {
		t.Errorf("No error on no name: %s", line)
	}

	line = []byte("somename:#|c")
	_, err = parseLine(line)
	if err == nil {
		t.Errorf("No error on invalid value: %s", line)
	}

	line = []byte("a_counter:1|k")
	_, err = parseLine(line)
	if err == nil {
		t.Errorf("No error on invalid type: %s", line)
	}

	line = []byte("somename:10.5|c|@0.1|#tagsandwhatnot|$wedontcare")
	e, err := parseLine(line)
	if err != nil {
		t.Errorf("Error when line contains unrecognized fields: Err: %s, Line: %s", err, line)
	}
	if e != nil {
		if 105.0 != e.Payload() {
			t.Errorf("SampleRate value incorrect: 105.0 != %f\n", e.Payload())
		}
	}

}

func TestStatsdParseLineCounter(t *testing.T) {
	line := []byte("my_counter:1|c")
	e, err := parseLine(line)
	if err != nil {
		t.Errorf("%s", err)
	}
	line = []byte("my_counter:1.05|c|@0.1")
	e2, err := parseLine(line)
	if err != nil {
		t.Errorf("%s", err)
	}
	if 10.5 != e2.Payload() {
		t.Errorf("SampleRate value incorrect: 10.5 != %f\n", e2.Payload())
	}
	e.Update(e2)
	if 11.5 != e.Payload() {
		t.Errorf("SampleRate value incorrect: 11.5 != %f\n", e.Payload())
	}
}