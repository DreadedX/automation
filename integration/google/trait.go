package google

import "github.com/kr/pretty"

type Trait string

// https://developers.google.com/assistant/smarthome/traits/onoff
const TraitOnOff Trait = "action.devices.traits.OnOff"

func (d *Device) AddOnOffTrait(onlyCommand bool, onlyQuery bool) *Device {
	d.Traits = append(d.Traits, TraitOnOff)
	if onlyCommand {
		d.Attributes["commandOnlyOnOff"] = true
	}
	if onlyQuery {
		d.Attributes["queryOnlyOnOff"] = true
	}

	return d
}

// https://developers.google.com/assistant/smarthome/traits/startstop
const TraitStartStop = "action.devices.traits.StartStop"

func (d *Device) AddStartStopTrait(pausable bool) *Device {
	d.Traits = append(d.Traits, TraitStartStop)

	if pausable {
		d.Attributes["pausable"] = true
	}

	return d
}

// https://developers.google.com/assistant/smarthome/traits/onoff
const TraitRunCycle = "action.devices.traits.RunCycle"

func (d *Device) AddRunCycleTrait() *Device {
	d.Traits = append(d.Traits, TraitRunCycle)

	return d
}

// https://developers.google.com/assistant/smarthome/traits/camerastream
const TraitCameraStream = "action.devices.traits.CameraStream"

func (d *Device) AddCameraStreamTrait(authTokenNeeded bool, supportedProtocols ...string) *Device {
	d.Traits = append(d.Traits, TraitCameraStream)

	if len(supportedProtocols) > 0 {
		d.Attributes["cameraStreamSupportedProtocols"] = supportedProtocols
	}

	d.Attributes["cameraStreamNeedAuthToken"] = authTokenNeeded

	pretty.Logln(d)

	return d
}

// https://developers.google.com/assistant/smarthome/traits/scene
const TraitScene = "action.devices.traits.Scene"

func (d *Device) AddSceneTrait(reversible bool) *Device {
	d.Traits = append(d.Traits, TraitScene)

	if reversible {
		d.Attributes["sceneReversible"] = true
	}

	return d
}
