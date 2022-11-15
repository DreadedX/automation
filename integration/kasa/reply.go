package kasa

type errCode struct {
	ErrCode int
}

type reply struct {
	System struct {
		SetRelayState errCode `json:"set_relay_state"`
	} `json:"system"`
}

type cmd struct {
	System struct {
		SetRelayState struct {
			State int `json:"state"`
		} `json:"set_relay_state"`
	} `json:"system"`
}
