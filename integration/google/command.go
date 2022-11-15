package google

import (
	"encoding/json"
	"fmt"
)

type CommandName string

type Execution struct {
	Name CommandName

	OnOff           *CommandOnOffData
	StartStop       *CommandStartStopData
	GetCameraStream *CommandGetCameraStreamData
	ActivateScene *CommandActivateSceneData
}

func (c *Execution) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name   CommandName     `json:"command"`
		Params json.RawMessage `json:"params,omitempty"`
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	c.Name = tmp.Name

	var details interface{}
	switch c.Name {
	case CommandOnOff:
		c.OnOff = &CommandOnOffData{}
		details = c.OnOff

	case CommandStartStop:
		c.StartStop = &CommandStartStopData{}
		details = c.StartStop

	case CommandGetCameraStream:
		c.GetCameraStream = &CommandGetCameraStreamData{}
		details = c.GetCameraStream

	case CommandActivateScene:
		c.ActivateScene = &CommandActivateSceneData{}
		details = c.ActivateScene

	default:
		return fmt.Errorf("Command (%s) is not implemented", c.Name)
	}

	err = json.Unmarshal(tmp.Params, details)
	if err != nil {
		return err
	}

	return nil
}

// https://developers.google.com/assistant/smarthome/traits/onoff
const CommandOnOff CommandName = "action.devices.commands.OnOff"

type CommandOnOffData struct {
	On bool `json:"on"`
}

// https://developers.google.com/assistant/smarthome/traits/startstop
const CommandStartStop CommandName = "action.devices.commands.StartStop"

type CommandStartStopData struct {
	Start         bool     `json:"start"`
	Zone          string   `json:"zone,omitempty"`
	MultipleZones []string `json:"multipleZones,omitempty"`
}

// https://developers.google.com/assistant/smarthome/traits/camerastream
const CommandGetCameraStream CommandName = "action.devices.commands.GetCameraStream"

type CommandGetCameraStreamData struct {
	StreamToChromecast       bool     `json:"StreamToChromecast"`
	SupportedStreamProtocols []string `json:"SupportedStreamProtocols"`
}

// https://developers.google.com/assistant/smarthome/traits/scene
const CommandActivateScene CommandName = "action.devices.commands.ActivateScene"

type CommandActivateSceneData struct {
	Deactivate bool `json:"deactivate"`
}
