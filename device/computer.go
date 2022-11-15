package device

import (
	"automation/integration/google"
	"log"
	"net/http"
)

type computer struct {
	macAddress string
	name string
	room string
}

func NewComputer(macAddress string, name string, room string) *computer {
	c := &computer{macAddress: macAddress, name: name, room: room}

	return c
}

func (c *computer) Sync() *google.Device {
	device := google.NewDevice(c.GetID(), google.TypeScene)
	device.AddSceneTrait(false)

	device.Name = google.DeviceName{
		DefaultNames: []string{
			"Computer",
		},
		Name: c.name,
	}
	device.RoomHint = c.room

	return device
}

func (c *computer) Query() google.DeviceState {
	state := google.NewDeviceState(true)
	state.Status = google.StatusSuccess

	return state
}

func (c *computer) Execute(execution google.Execution, updateState *google.DeviceState) (string, bool) {
	errCode := ""

	switch execution.Name {
	case google.CommandActivateScene:
		if !execution.ActivateScene.Deactivate {
			// @NOTE For now just call the webhook, that way we do not have to give this docker container network_mode: host
			// @TODO Add wake on lan support

			http.Get("https://webhook.huizinga.dev/start-pc?token=7!$8bmjfZsT606Rmw5IrfIXhQWt6clTY")
		}
	default:
		errCode = "actionNotAvailable"
		log.Printf("Command (%s) not supported\n", execution.Name)
	}

	return errCode, true
}

func (c *computer) GetID() string {
	return c.macAddress
}
