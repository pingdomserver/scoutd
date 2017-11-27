package statsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pingdomserver/scoutd/collectors/event"
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

type StatsdCollector struct {
	name           string
	addr           string
	flushInterval  time.Duration
	eventLimit     int
	eventChannel   chan event.Event
	events         map[string]event.Event
	eventsSnapshot map[string]event.Event
	messageChannel chan CollectorMessage
	eventBlacklist map[string]time.Time
	eventsRcvd     int64
	eventsDropped  int64
	pktsRcvd       int64
	pktParseErrs   int64
	pktReadErrs    int64
	badPackets     int64
}


const (
	DefaultStatsdAddr = "127.0.0.1:8125"
	DefaultEventLimit = 1000
)

func NewStatsdCollector(name string, addr string, flushInterval time.Duration, eventLimit int) (*StatsdCollector, error) {
	if name == "" {
		return nil, fmt.Errorf("collector name cannot be empty")
	}
	sd := &StatsdCollector{
		name:           name,
		addr:           addr,
		flushInterval:  flushInterval,
		eventLimit:     eventLimit,
		eventChannel:   make(chan event.Event, 100),
		events:         make(map[string]event.Event, 0),
		eventsSnapshot: make(map[string]event.Event, 0),
		messageChannel: make(chan CollectorMessage, 10),
		eventBlacklist: make(map[string]time.Time, 0),
	}
	return sd, nil
}

// Starts the statsd aggregator and the UDP socket listener.
// You must call Start() before this StatsdCollector will
// begin listening for, and aggregating, statsd packets.
func (sd *StatsdCollector) Start() {
	defer func() {
		if r := recover(); r != nil {
			//sd.Shutdown()  TODO: shutdown clean on panic
			fmt.Printf("panic in statsd: %s", r)
		}
	}()

	go sd.aggregate()
	go sd.ListenAndReceive()
}

// The central aggregator for the StatsdCollector.
// It is crucial to handle both the flushing/snapshotting and event updates synchronously.
// All events are processed from the sd.eventChannel to avoid locking the sd.Events map
func (sd *StatsdCollector) aggregate() {
	defer func(sd *StatsdCollector) {
		if r := recover(); r != nil {
			fmt.Println("Caught panic in aggregate")
			panic(r)
		}
	}(sd)

	flushTicker := time.NewTicker(sd.flushInterval)
	//pktRcvd := 0
	for {
		select {
		case <-flushTicker.C:
			sd.eventsSnapshot = make(map[string]event.Event, len(sd.events))
			for k, e := range sd.events {
				if _, blacklisted := sd.eventBlacklist[k]; blacklisted {
					continue // go to next event in for/range
				}
				sd.eventsSnapshot[k] = e.Copy()
				switch e.Type() {
				case event.EventIncr, event.EventTiming:
					e.Reset()
				}
			}
			if len(sd.eventsSnapshot) > 0 {
				sd.eventsSnapshot["statsd.events_total"] = &event.Increment{Name: "statsd.events_total", Value: float64(len(sd.eventsSnapshot))}
				sd.eventsSnapshot["statsd.events_received"] = &event.Increment{Name: "statsd.events_received", Value: float64(sd.eventsRcvd)}
				// Disable reorting of these internal statsd metrics for now.
				//sd.eventsSnapshot["statsd.events_dropped"] = &event.Increment{Name: "statsd.events_dropped", Value: float64(sd.eventsDropped)}
				//sd.eventsSnapshot["statsd.packets_received"] = &event.Increment{Name: "statsd.packets_received", Value: float64(sd.pktsRcvd)}
				//sd.eventsSnapshot["statsd.packet_read_errors"] = &event.Increment{Name: "statsd.packet_read_errors", Value: float64(sd.pktReadErrs)}
				//sd.eventsSnapshot["statsd.packet_parse_errors"] = &event.Increment{Name: "statsd.packet_parse_errors", Value: float64(sd.pktParseErrs)}
				//sd.eventsSnapshot["statsd.bad_packets"] = &event.Increment{Name: "statsd.bad_packets", Value: float64(sd.badPackets)}
			}
			sd.eventsRcvd = 0
			sd.eventsDropped = 0
			sd.pktsRcvd = 0
			sd.pktParseErrs = 0
			sd.pktReadErrs = 0
			sd.badPackets = 0
			sd.eventBlacklist = make(map[string]time.Time, 0)
		case e := <-sd.eventChannel:
			sd.eventsRcvd += 1
			// The events are stored in a map keyed by the metric name.
			// Any operations on the metric namespace should be done here so that we update the
			// correct event.
			k := e.Key()
			e.SetKey(k)

			if _, blacklisted := sd.eventBlacklist[k]; blacklisted {
				sd.eventsDropped += 1
				break // break back to the "for" loop
			}

			if e2, ok := sd.events[k]; ok {
				// Update an existing event
				e2.Update(e)
				sd.events[k] = e2
			} else {
				if len(sd.events) < sd.eventLimit {
					// Add a new event
					sd.events[k] = e
				} else {
					sd.eventsDropped += 1
				}
			}
		case msg := <-sd.messageChannel:
			sd.processCollectorMessage(msg)
			//case c := <-sb.closeChannel:
			//	Flush before closing
			//	c.reply <- sb.flush()
			//	return
		}
	}
}

// Collect() is a noop method for a statsdCollector.
func (sd *StatsdCollector) Collect() error {
	return nil
}

// Set up the UDP listener socket, pass conn to sd.Receive()
func (sd *StatsdCollector) ListenAndReceive() error {
	addr := sd.addr
	if addr == "" {
		addr = DefaultStatsdAddr
	}
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	return sd.Receive(conn)
}

// Handles the reading of the UDP packet. Sends the contents of the UDP packet to sd.handleMessage()
func (sd *StatsdCollector) Receive(conn net.PacketConn) error {
	defer conn.Close()

	msg := make([]byte, 1024)
	for {
		nbytes, addr, err := conn.ReadFrom(msg)
		if err != nil {
			log.Printf("%s", err)
			sd.pktReadErrs += 1
			continue
		}
		buf := make([]byte, nbytes)
		copy(buf, msg[:nbytes])
		sd.pktsRcvd += 1
		go sd.handleMessage(addr, buf)
	}
	panic("error reading from udp socket")
}

// Handles the contents of a message received from Receive()
// Reads each line of the message and sends to parseLine()
// On parseLine() success, we get beck an event.Event and send it to sd.eventChannel
func (sd *StatsdCollector) handleMessage(addr net.Addr, msg []byte) {
	buf := bytes.NewBuffer(msg)
	for {
		line, readerr := buf.ReadBytes('\n')

		// protocol does not require line to end in \n, if EOF use received line if valid
		if readerr != nil && readerr != io.EOF {
			//log.Printf("error reading message from %s: %s", addr, readerr)
			sd.badPackets += 1
			return
		} else if readerr != io.EOF {
			// remove newline, only if not EOF
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
		}

		// Only process lines with more than one character
		if len(line) > 1 {
			evnt, err := parseLine(line)
			if err != nil {
				// Log the error
				//fmt.Printf("Parsing error: %s", err)
				sd.pktParseErrs += 1
				return
			}
			sd.eventChannel <- evnt
		}

		if readerr == io.EOF {
			return // done with this message
		}
	}
}

// Parses a single line in statsd protocol format and returns an event.Event
func parseLine(line []byte) (event.Event, error) {
	var err error
	fieldSep := []byte("|")
	valSep := []byte(":")

	s := bytes.Split(line, fieldSep)
	if len(s) < 2 {
		return nil, fmt.Errorf("error parsing metric: not enough fields")
	}

	nameAndVal := s[0]
	typeString := string(s[1])

	valIndex := bytes.Index(nameAndVal, valSep)
	if valIndex <= 0 {
		return nil, fmt.Errorf("error parsing metric: invalid name")
	}
	name := string(nameAndVal[:valIndex])

	if len(nameAndVal) - (valIndex+1) < 1 {
		return nil, fmt.Errorf("error parsing metric: no value")
	}
	value, err := strconv.ParseFloat(string(nameAndVal[valIndex+1:]), 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric: invalid value")
	}

	var sampleRate float64
	sampleRate = 1.0

	if len(s) > 2 {
		for _, fieldBytes := range s[2:] {
			if len(fieldBytes) > 1 {
				switch string(fieldBytes[0]) {
				case "@":
					rateValue, _ := strconv.ParseFloat(string(fieldBytes[1:]), 64)
					if rateValue > 0.0 && rateValue <= 1.0 {
						sampleRate = rateValue
					}
				}
			}
		}
	}

	var evnt event.Event

	switch typeString {
	case "ms":
		// Timer
		evnt = event.NewTiming(name, float64(value))
	case "g":
		// Gauge
		evnt = &event.Gauge{Name: name, Value: float64(value)}
	case "c":
		// Counter
		evnt = &event.Increment{Name: name, Value: float64(value), SampleRate: sampleRate}
	default:
		err = fmt.Errorf("invalid metric type: %q", typeString)
		return nil, err
	}

	return evnt, nil
}

// Calculates each event of sd.EventsSnapshot into a Metric struct
// Returns a pointer to a CollectorPayload to prevent copy overhead
func (sd *StatsdCollector) Payload() *CollectorPayload {
	metrics := []*event.Metric{}
	for k, e := range sd.eventsSnapshot {
		if _, blacklisted := sd.eventBlacklist[k]; blacklisted {
			continue // go to next event in for/range
		}
		for _, m := range e.Metrics() {
			metrics = append(metrics, m)
		}
    e.Reset() // reset Event data
	}
	payload := &CollectorPayload{
		Name:    sd.name,
		Type:    sd.TypeString(),
		Metrics: metrics,
	}
	return payload
}

func (sd *StatsdCollector) processCollectorMessage(msg CollectorMessage) {
	switch msg.MessageType {
	case "delete_metrics":
		metricNames := []string{}
		err := json.Unmarshal(msg.Data, &metricNames)
		if err != nil {
			log.Printf("Error unmarshalling metric names: %s\n", err)
		}
		now := time.Now()
		for _, name := range metricNames {
			sd.eventBlacklist[name] = now.UTC()
			delete(sd.events, name)
		}
	}
}

func (sd *StatsdCollector) ReceiveCollectorMessage(msg CollectorMessage) {
	switch msg.MessageType {
	case "delete_metrics":
		sd.messageChannel <- msg
	}
}

// Returns sd.name
func (sd *StatsdCollector) Name() string {
	return sd.name
}

// Returns the integer constant of this Collector type
func (sd *StatsdCollector) Type() int {
	return StatsdType
}

// Returns a string of this Collector type
func (sd *StatsdCollector) TypeString() string {
	return "statsd"
}
