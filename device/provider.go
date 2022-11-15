package smarthome

import (
	"automation/integration/google"
	"automation/integration/mqtt"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/kr/pretty"
	"google.golang.org/api/homegraph/v1"
	"google.golang.org/api/option"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInfo struct {
	IEEEAdress string `json:"ieee_address"`
	FriendlyName string `json:"friendly_name"`
	Description string `json:"description"`
	Manufacturer string `json:"manufacturer"`
	ModelID string `json:"model_id"`
	SoftwareBuildID string `json:"software_build_id"`
}

type Provider struct {
	service *google.Service
	userID string

	devices map[string]google.DeviceInterface
}

func NewService(m *mqtt.MQTT) *google.Service {
	credentials64, _ := os.LookupEnv("GOOGLE_CREDENTIALS")
	credentials, err := base64.StdEncoding.DecodeString(credentials64)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	provider := &Provider{userID: "Dreaded_X", devices: make(map[string]google.DeviceInterface)}

	homegraphService, err := homegraph.NewService(context.Background(), option.WithCredentialsJSON(credentials))
	if err != nil {
		panic(err)
	}

	provider.service = google.NewService(provider, homegraphService)

	m.AddHandler("zigbee2mqtt/bridge/devices", func(_ paho.Client, msg paho.Message) {
		var devices []DeviceInfo
		json.Unmarshal(msg.Payload(), &devices)

		log.Println("zigbee2mqtt devices:")
		pretty.Logln(devices)

		// Clear the list of devices in order to update it
		provider.devices = make(map[string]google.DeviceInterface)
		for _, device := range devices {
			switch device.Description {
			case "Kettle":
				outlet := NewKettle(device, m, provider.service)
				provider.devices[device.IEEEAdress] = outlet
				log.Printf("Added Kettle (%s) %s\n", device.IEEEAdress, device.FriendlyName)
			}
		}

		// Send sync request
		provider.service.RequestSync(context.Background(), provider.userID)
	})

	return provider.service
}

func (p *Provider) Sync(_ context.Context, _ string) ([]*google.Device, error) {
	var devices []*google.Device

	for _, device := range p.devices {
		devices = append(devices, device.Sync())
	}

	return devices, nil
}

func (p *Provider) Query(_ context.Context, _ string, handles []google.DeviceHandle) (map[string]google.DeviceState, error) {
	states := make(map[string]google.DeviceState)

	for _, handle := range handles {
		if device, found := p.devices[handle.ID]; found {
			states[handle.ID] = device.Query()
		} else {
			log.Printf("Device (%s) not found\n", handle.ID)
		}
	}

	return states, nil
}

func (p *Provider) Execute(_ context.Context, _ string, commands []google.Command) (*google.ExecuteResponse, error) {
	resp := &google.ExecuteResponse{
		UpdatedState: google.NewDeviceState(true),
		FailedDevices: make(map[string]struct{Devices []string}),
	}

	for _, command := range commands {
		for _, execution := range command.Execution {
			for _, handle := range command.Devices {
				if device, found := p.devices[handle.ID]; found {
					errCode, online := device.Execute(execution, &resp.UpdatedState)

					// Update the state
					p.devices[handle.ID] = device
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
					log.Printf("Device (%s) not found\n", handle.ID)
				}
			}
		}
	}

	return resp, nil
}
