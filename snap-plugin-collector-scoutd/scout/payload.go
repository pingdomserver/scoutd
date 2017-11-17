package scout

import (
  "os"
  "bufio"
  "fmt"
)

type AgentPayload struct {
  ClientData interface{}
  StatsdData interface{}
}

func ToPayload(payload AgentPayload) interface{} {
  return payload.ClientData;
}

func SavePayload(payload AgentPayload) {
  f, _ := os.Create("/tmp/scout")

  w := bufio.NewWriter(f)
  data := ToPayload(payload)

  fmt.Fprintf(w, "%v\n", data)

  w.Flush()
}
