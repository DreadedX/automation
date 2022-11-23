package zigbee

import (
	"automation/device"
	"automation/integration/google"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type kettle struct {
	info Info

	client paho.Client
	service *google.Service

	updated chan bool

	isOn bool
	online bool
}

func NewKettle(info Info, client paho.Client, service *google.Service) *kettle {
	k := &kettle{info: info, client: client, service: service, updated: make(chan bool, 1)}

	if token := k.client.Subscribe(k.info.MQTTAddress, 1, k.stateHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	return k
}

func (k *kettle) stateHandler(client paho.Client, msg paho.Message) {
	var payload OnOffState
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Println(err)
		return
	}

	// Update the internal state
	k.isOn = payload.State
	k.online = true

	// Notify that the state has updated
	for len(k.updated) > 0 {
		<- k.updated
	}
	k.updated <- true

	// Notify google of the updated state
	id := k.GetID().String()
	k.service.ReportState(context.Background(), id, map[string]google.DeviceState{
		id: k.getState(),
	})
}

func (k *kettle) getState() google.DeviceState {
	return google.NewDeviceState(k.online).RecordOnOff(k.isOn)
}


// zigbee.Device
var _ Device = (*kettle)(nil)
func (k *kettle) IsZigbeeDevice() {}

// zigbee.Device
func (k *kettle) Delete(client paho.Client) {
	if token := client.Unsubscribe(k.info.MQTTAddress); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

// google.DeviceInterface
var _ google.DeviceInterface = (*kettle)(nil)
func (*kettle) IsGoogleDevice() {}

// google.DeviceInterface
func (k *kettle) Sync() *google.Device {
	device := google.NewDevice(k.GetID().String(), google.TypeKettle)
	device.AddOnOffTrait(false, false)

	device.Name = google.DeviceName{
		DefaultNames: []string{
			"Kettle",
		},
		Name: strings.Title(k.GetID().Name()),
	}

	device.WillReportState = true
	room := strings.Title(k.GetID().Room())
	if len(room) > 1 {
		device.RoomHint = room
	}

	device.DeviceInfo = google.DeviceInfo{
		Manufacturer: k.info.Manufacturer,
		Model: k.info.ModelID,
		SwVersion: k.info.SoftwareBuildID,
	}

	return device
}

// google.DeviceInterface
func (k *kettle) Query() google.DeviceState {
	state := k.getState()
	if k.online {
		state.Status = google.StatusSuccess
	} else {
		state.Status = google.StatusOffline
	}

	return state
}

// google.DeviceInterface
func (k *kettle) Execute(execution google.Execution, updatedState *google.DeviceState) (string, bool) {
	errCode := ""

	switch execution.Name {
	case google.CommandOnOff:

		// Clear the updated channel
		for len(k.updated) > 0 {
			<- k.updated
		}

		k.SetOnOff(execution.OnOff.On)

		// Start timeout timer
		timer := time.NewTimer(time.Second)

		// Wait for the update or timeout
		select {
			case <- k.updated:
				updatedState.RecordOnOff(k.isOn)

			case <- timer.C:
				// If we do not get a response in time mark the device as offline
				log.Println("Device did not respond, marking as offline")
				k.online = false
		}

	default:
		// @TODO Should probably move the error codes to a enum
		errCode = "actionNotAvailable"
		log.Printf("Command (%s) not supported\n", execution.Name)
	}

	return errCode, k.online
}

// device.Base
var _ device.Basic = (*kettle)(nil)
func (k *kettle) GetID() device.InternalName {
	return k.info.FriendlyName
}

// device.OnOff
var _ device.OnOff = (*kettle)(nil)
func (k *kettle) SetOnOff(state bool) {
	msg := "OFF"
	if state {
		msg = "ON"
	}

	if token := k.client.Publish(fmt.Sprintf("%s/set", k.info.MQTTAddress), 1, false, fmt.Sprintf(`{ "state": "%s" }`, msg)); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

// device.OnOff
func (k *kettle) GetOnOff() bool {
	return k.isOn
}
