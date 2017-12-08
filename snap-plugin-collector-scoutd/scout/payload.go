package scout

import (
	"encoding/json"

	"github.com/pingdomserver/scoutd/collectors"
)

type ScoutPayload struct {
	StatsDPayload      []*collectors.CollectorPayload
	ScoutClientPayload json.RawMessage
}
