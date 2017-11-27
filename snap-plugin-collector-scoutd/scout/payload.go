package scout

import (
  "encoding/json"
  "./statsd"
)

type ScoutPayload struct {
  StatsDPayload []*statsd.CollectorPayload
  ScoutClientPayload json.RawMessage
}
