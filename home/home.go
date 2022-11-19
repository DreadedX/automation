package home

import (
	"automation/config"
	"automation/device"
	"automation/integration/google"
	"context"
	"log"

	"google.golang.org/api/homegraph/v1"
	"google.golang.org/api/option"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Home struct {
	Service *google.Service
	Username  string

	Devices map[device.InternalName]device.Basic
}

// Auto populate and update the device list
func New(username string, credentials config.Credentials, client paho.Client) *Home {
	home := &Home{Username: username, Devices: make(map[device.InternalName]device.Basic)}

	homegraphService, err := homegraph.NewService(context.Background(), option.WithCredentialsJSON(credentials))
	if err != nil {
		panic(err)
	}

	home.Service = google.NewService(home, homegraphService)

	return home
}

func (h *Home) AddDevice(d device.Basic) {
	h.Devices[d.GetID()] = d

	log.Printf("Added %s in %s (%s)\n", d.GetID().Name(), d.GetID().Room(), d.GetID())
}

func (h *Home) Sync(_ context.Context, _ string) ([]*google.Device, error) {
	var devices []*google.Device

	for _, device := range device.GetDevices[google.DeviceInterface](&h.Devices) {
		devices = append(devices, device.Sync())
	}

	return devices, nil
}

func (h *Home) Query(_ context.Context, _ string, handles []google.DeviceHandle) (map[string]google.DeviceState, error) {
	states := make(map[string]google.DeviceState)

	for _, handle := range handles {
		if device, err := device.GetDevice[google.DeviceInterface](&h.Devices, device.InternalName(handle.ID)); err == nil {
			states[device.GetID().String()] = device.Query()
		} else {
			log.Println(err)
		}
	}

	return states, nil
}

func (h *Home) Execute(_ context.Context, _ string, commands []google.Command) (*google.ExecuteResponse, error) {
	resp := &google.ExecuteResponse{
		UpdatedState:  google.NewDeviceState(true),
		FailedDevices: make(map[string]struct{ Devices []string }),
	}

	for _, command := range commands {
		for _, execution := range command.Execution {
			for _, handle := range command.Devices {
				if device, err := device.GetDevice[google.DeviceInterface](&h.Devices, device.InternalName(handle.ID)); err == nil {
					errCode, online := device.Execute(execution, &resp.UpdatedState)

					// Update the state
					h.Devices[device.GetID()] = device
					if !online {
						resp.OfflineDevices = append(resp.OfflineDevices, handle.ID)
					} else if len(errCode) == 0 {
						resp.UpdatedDevices = append(resp.UpdatedDevices, handle.ID)
					} else {
						e := resp.FailedDevices[errCode]
						e.Devices = append(e.Devices, handle.ID)
						resp.FailedDevices[errCode] = e
					}
				} else {
					log.Println(err)
				}
			}
		}
	}

	return resp, nil
}
