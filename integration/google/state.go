package google

import (
	"encoding/json"
)

type DeviceState struct {
	Online bool
	Status Status

	state map[string]interface{}
}

func (ds DeviceState) MarshalJSON() ([]byte, error) {
	payload := make(map[string]interface{})

	payload["online"] = ds.Online
	if len(ds.Status) > 0 {
		payload["status"] = ds.Status
	}

	for k, v := range ds.state {
		payload[k] = v
	}

	return json.Marshal(payload)
}

func NewDeviceState(online bool) DeviceState {
	return DeviceState{
		Online: online,
		state: make(map[string]interface{}),
	}
}

// https://developers.google.com/assistant/smarthome/traits/onoff
func (ds DeviceState) RecordOnOff(on bool) DeviceState {
	ds.state["on"] = on

	return ds
}

// https://developers.google.com/assistant/smarthome/traits/runcycle
func (ds DeviceState) RecordRunCycle(state int) DeviceState {
	if state == 0 {
	} else if state == 1 {
		ds.state["currentRunCycle"] = []struct{
			CurrentCycle string `json:"currentCycle"`
			Lang string `json:"lang"`
		}{
			{
				CurrentCycle: "Wash",
				Lang: "en",
			},
		}
	} else if state == 2 {
		ds.state["currentTotalRemainingTime"] = 0
	}

	return ds
}

// https://developers.google.com/assistant/smarthome/traits/startstop
func (ds DeviceState) RecordStartStop(running bool, paused ...bool) DeviceState {
	ds.state["isRunning"] = running
	if len(paused) > 0 {
		ds.state["isPaused"] = paused[0]
	}

	return ds
}

// https://developers.google.com/assistant/smarthome/traits/camerastream
func (ds DeviceState) RecordCameraStream(url string) DeviceState {
	ds.state["cameraStreamProtocol"] = "progressive_mp4"
	ds.state["cameraStreamAccessUrl"] = url

	return ds
}
