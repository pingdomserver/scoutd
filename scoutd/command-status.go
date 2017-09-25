package scoutd

type StatusOptions struct {
	// empty for now
}

var statusOptions StatusOptions

func init() {
	parser.AddCommand("status", "Check to see if scoutd is running and to verify configuration options", "", &statusOptions)
}
