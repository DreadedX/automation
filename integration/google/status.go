package google

type Status string

const (
	StatusSuccess   Status = "SUCCESS"
	StatusOffline          = "OFFLINE"
	StatusException        = "EXCEPTIONS"
	StatusError            = "ERROR"
)
