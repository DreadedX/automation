package kasa

type reply struct {
	System struct {
		SetRelayState struct {
			ErrCode int `json:"err_code"`
		} `json:"set_relay_state"`
		GetSysinfo struct {
			RelayState int `json:"relay_state"`
			ErrCode int `json:"err_code"`
		} `json:"get_sysinfo"`
	} `json:"system"`
}

type SetRelayState struct {
	State int `json:"state"`
}

type GetSysinfo struct{}

type cmd struct {
	System struct {
		SetRelayState *SetRelayState `json:"set_relay_state,omitempty"`
		GetSysinfo    *GetSysinfo    `json:"get_sysinfo,omitempty"`
	} `json:"system"`
}
