package zigbee

import "encoding/json"

type OnOffState struct {
	State bool
}

func (k *OnOffState) UnmarshalJSON(data []byte) error {
	var payload struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	k.State = payload.State == "ON"

	return nil
}

type RemoteAction string

const (
	ACTION_ON              RemoteAction = "on"
	ACTION_OFF                          = "off"
	ACTION_BRIGHTNESS_UP                = "brightness_move_up"
	ACTION_BRIGHTNESS_DOWN              = "brightness_move_down"
	ACTION_BRIGHTNESS_STOP              = "brightness_move_down"
)

type RemoteState struct {
	Action RemoteAction `json:"action"`
}
