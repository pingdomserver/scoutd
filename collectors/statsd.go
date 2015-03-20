package collectors

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"
	"github.com/scoutapp/scoutd/collectors/event"
)

const (
	DefaultStatsdAddr = "127.0.0.1:8125"
)

type StatsdCollector struct {
	name           string
	addr           string
	flushInterval  time.Duration
	eventChannel   chan event.Event
	events         map[string]event.Event
	eventsSnapshot map[string]event.Event
}

// Initializes a new StatsdCollector. You must call Start() before this StatsdCollector will
// begin listening for, and aggregating, statsd packets.
func NewStatsdCollector(name string, flushInterval time.Duration) (*StatsdCollector, error){
	if name == "" {
		return nil, fmt.Errorf("collector name cannot be empty")
	}
	sd := &StatsdCollector{
		name:           name,
		flushInterval:  flushInterval,
		eventChannel:   make(chan event.Event, 100),
		events:         make(map[string]event.Event, 0),
		eventsSnapshot: make(map[string]event.Event, 0),
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
			sd.eventsSnapshot = sd.events
			sd.events = make(map[string]event.Event, 0)
			//fmt.Printf("Pkts: %d \n Metrics: %+v\n", pktRcvd, len(sd.eventsSnapshot))
			//pktRcvd = 0
		case e := <-sd.eventChannel:
			//pktRcvd += 1
			// The events are stored in a map keyed by the metric name.
			// Any operations on the metric namespace should be done here so that we update the
			// correct event.
			k := e.Key()
			e.SetKey(k)

			if e2, ok := sd.events[k]; ok {
				// Update an existing event
				e2.Update(e)
				sd.events[k] = e2
			} else {
				// Add a new event
				sd.events[k] = e
			}
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
			continue
		}
		buf := make([]byte, nbytes)
		copy(buf, msg[:nbytes])
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
			log.Printf("error reading message from %s: %s", addr, readerr)
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
				fmt.Printf("Parsing error: %s", err)
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
	buf := bytes.NewBuffer(line)
	bname, err := buf.ReadBytes(':')
	if err != nil {
		return nil, fmt.Errorf("error parsing metric name: %s", err)
	}
	name := string(bname[:len(bname)-1])

	bvalue, err := buf.ReadBytes('|')
	if err != nil {
		return nil, fmt.Errorf("error parsing metric value: %s", err)
	}
	value, err := strconv.ParseFloat(string(bvalue[:len(bvalue)-1]), 64)
	if err != nil {
		return nil, fmt.Errorf("error converting metric value: %s", err)
	}

	metricType := buf.Bytes()
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error parsing metric type: %s", err)
	}

	var evnt event.Event

	switch string(metricType[:len(metricType)]) {
	case "ms":
		// Timer
		evnt = event.NewTiming(name, float64(value))
	case "g":
		// Gauge
		evnt = &event.Gauge{Name: name, Value: float64(value)}
	case "c":
		// Counter
		evnt = &event.Increment{Name: name, Value: float64(value)}
	default:
		err = fmt.Errorf("invalid metric type: %q", metricType)
		return nil, err
	}

	return evnt, nil
}

// Calculates each event of sd.EventsSnapshot into a Metric struct
// Returns a pointer to a CollectorPayload to prevent copy overhead
func (sd *StatsdCollector) Payload() *CollectorPayload {
	metrics := []*event.Metric{}
	for _, e := range sd.eventsSnapshot {
		for _, m := range e.Metrics(){
			metrics = append(metrics, m)
		}
	}
	payload := &CollectorPayload{
		Name:    sd.name,
		Type:    sd.TypeString(),
		Metrics: metrics,
	}
	return payload
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
