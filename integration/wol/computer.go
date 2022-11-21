package wol

import (
	"automation/device"
	"automation/integration/google"
	"log"
	"net/http"
)

type computer struct {
	macAddress string
	name device.InternalName
	url string
}

func NewComputer(macAddress string, name device.InternalName, url string) *computer {
	c := &computer{macAddress, name, url}

	return c
}

func (c *computer) Activate(state bool) {
	if state {
		_, err := http.Get(c.url)
		if err != nil {
			log.Println(err)
		}
	} else {
		// This is not implemented
	}
}

// device.Basic
var _ device.Basic = (*computer)(nil)
func (c *computer) GetID() device.InternalName {
	return device.InternalName(c.name)
}

// google.DeviceInterface
var _ google.DeviceInterface = (*computer)(nil)
func (*computer) IsGoogleDevice() {}

// google.DeviceInterface
func (c *computer) Sync() *google.Device {
	device := google.NewDevice(c.GetID().String(), google.TypeScene)
	device.AddSceneTrait(false)

	device.Name = google.DeviceName{
		DefaultNames: []string{
			"Computer",
		},
		Name: c.GetID().Name(),
	}
	device.RoomHint = c.GetID().Room()

	return device
}

// google.DeviceInterface
func (c *computer) Query() google.DeviceState {
	state := google.NewDeviceState(true)
	state.Status = google.StatusSuccess

	return state
}

// google.DeviceInterface
func (c *computer) Execute(execution google.Execution, updateState *google.DeviceState) (string, bool) {
	errCode := ""

	switch execution.Name {
	case google.CommandActivateScene:
		c.Activate(!execution.ActivateScene.Deactivate)
	default:
		errCode = "actionNotAvailable"
		log.Printf("Command (%s) not supported\n", execution.Name)
	}

	return errCode, true
}
