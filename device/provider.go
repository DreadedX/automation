package device

import (
	"automation/integration/google"
	"automation/integration/kasa"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/kr/pretty"
	"google.golang.org/api/homegraph/v1"
	"google.golang.org/api/option"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type BaseDevice interface {
	GetName() string
}

type Devices struct {
	Devices map[string]interface{}
}

func NewDevices() *Devices {
	return &Devices{Devices: make(map[string]interface{})}
}

func (d *Devices) GetGoogleDevices() map[string]google.DeviceInterface {
	devices := make(map[string]google.DeviceInterface)

	for _, device := range d.Devices {
		if gd, ok := device.(google.DeviceInterface); ok {
			// Instead of using name we use the internal ID for google, that way devices can freely be renamed without causing issues with google home
			devices[gd.GetID()] = gd
		}
	}

	return devices
}

func (d *Devices) GetGoogleDevice(name string) (google.DeviceInterface, error) {
	device, ok := d.GetGoogleDevices()[name]
	if !ok {
		return nil, fmt.Errorf("Device does not exist")
	}

	return device, nil
}

func (d *Devices) GetZigbeeDevices() map[string]ZigbeeDevice {
	devices := make(map[string]ZigbeeDevice)

	for name, device := range d.Devices {
		if zd, ok := device.(ZigbeeDevice); ok {
			devices[name] = zd
		}
	}

	return devices
}

func (d *Devices) GetKasaDevices() map[string]*kasa.Kasa {
	devices := make(map[string]*kasa.Kasa)

	for _, device := range d.Devices {
		if gd, ok := device.(*kasa.Kasa); ok {
			// Instead of using name we use the internal ID for google, that way devices can freely be renamed without causing issues with google home
			devices[gd.GetName()] = gd
		}
	}

	return devices
}

func (d *Devices) GetKasaDevice(name string) (*kasa.Kasa, error) {
	device, ok := d.GetKasaDevices()[name]
	if !ok {
		return nil, fmt.Errorf("Device does not exist")
	}

	return device, nil
}

type DeviceInfo struct {
	IEEEAdress      string `json:"ieee_address"`
	FriendlyName    string `json:"friendly_name"`
	Description     string `json:"description"`
	Manufacturer    string `json:"manufacturer"`
	ModelID         string `json:"model_id"`
	SoftwareBuildID string `json:"software_build_id"`
}

type ZigbeeDevice interface {
	GetDeviceInfo() DeviceInfo
	SetState(state bool)
}

type DeviceInterface interface {
	google.DeviceInterface
	SetState(state bool)
}

type Provider struct {
	Service *google.Service
	userID  string

	Devices *Devices
}

type credentials []byte
type Config struct {
	Credentials credentials `yaml:"credentials" envconfig:"GOOGLE_CREDENTIALS"`
}

func (c *credentials) Decode(value string) error {
	b, err := base64.StdEncoding.DecodeString(value)
	*c = b

	return err
}

// Auto populate and update the device list
func (p *Provider) devicesHandler(client paho.Client, msg paho.Message) {
	var devices []DeviceInfo
	json.Unmarshal(msg.Payload(), &devices)

	log.Println("zigbee2mqtt devices:")
	pretty.Logln(devices)

	for name := range p.Devices.GetZigbeeDevices() {
		// Delete all zigbee devices from the device list
		delete(p.Devices.Devices, name)
	}

	for _, device := range devices {
		switch device.Description {
		case "Kettle":
			kettle := NewKettle(device, client, p.Service)
			p.Devices.Devices[kettle.GetDeviceInfo().FriendlyName] = kettle
			log.Printf("Added Kettle (%s) %s\n", kettle.GetDeviceInfo().IEEEAdress, kettle.GetDeviceInfo().FriendlyName)
		}
	}

	// Send sync request
	p.Service.RequestSync(context.Background(), p.userID)
}

func NewProvider(config Config, client paho.Client) *Provider {
	provider := &Provider{userID: "Dreaded_X", Devices: NewDevices()}

	homegraphService, err := homegraph.NewService(context.Background(), option.WithCredentialsJSON(config.Credentials))
	if err != nil {
		panic(err)
	}

	provider.Service = google.NewService(provider, homegraphService)

	if token := client.Subscribe("zigbee2mqtt/bridge/devices", 1, provider.devicesHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	return provider
}

func (p *Provider) AddDevice(device BaseDevice) {
	p.Devices.Devices[device.GetName()] = device
}

func (p *Provider) Sync(_ context.Context, _ string) ([]*google.Device, error) {
	var devices []*google.Device

	for _, device := range p.Devices.GetGoogleDevices() {
		devices = append(devices, device.Sync())
	}

	return devices, nil
}

func (p *Provider) Query(_ context.Context, _ string, handles []google.DeviceHandle) (map[string]google.DeviceState, error) {
	states := make(map[string]google.DeviceState)

	for _, handle := range handles {
		if device, err := p.Devices.GetGoogleDevice(handle.ID); err == nil {
			states[handle.ID] = device.Query()
		} else {
			log.Println(err)
		}
	}

	return states, nil
}

func (p *Provider) Execute(_ context.Context, _ string, commands []google.Command) (*google.ExecuteResponse, error) {
	resp := &google.ExecuteResponse{
		UpdatedState:  google.NewDeviceState(true),
		FailedDevices: make(map[string]struct{ Devices []string }),
	}

	for _, command := range commands {
		for _, execution := range command.Execution {
			for _, handle := range command.Devices {
				if device, err := p.Devices.GetGoogleDevice(handle.ID); err == nil {
					errCode, online := device.Execute(execution, &resp.UpdatedState)

					// Update the state
					p.Devices.Devices[handle.ID] = device
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
