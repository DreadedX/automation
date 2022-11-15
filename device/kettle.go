package smarthome

import (
	"automation/integration/mqtt"
	"automation/integration/google"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type outlet struct {
	Info DeviceInfo
	m *mqtt.MQTT
	updated chan bool

	isOn bool
	online bool
}

func (o *outlet) getState() google.DeviceState {
	return google.NewDeviceState(o.online).RecordOnOff(o.isOn)
}

func NewKettle(info DeviceInfo, m *mqtt.MQTT, s *google.Service) *outlet {
	o := &outlet{Info: info, m: m, updated: make(chan bool, 1)}

	const length = 5 * time.Minute
	timer := time.NewTimer(length)
	timer.Stop()

	go func() {
		for {
			<- timer.C
			log.Println("Turning kettle automatically off")
			m.Publish("zigbee2mqtt/kitchen/kettle/set", 1, false, `{"state": "OFF"}`)
		}
	}()

	o.m.AddHandler(fmt.Sprintf("zigbee2mqtt/%s", o.Info.FriendlyName), func (_ paho.Client, msg paho.Message)  {
		var payload struct {
			State string `json:"state"`
		}
		json.Unmarshal(msg.Payload(), &payload)

		// Update the internal state
		o.isOn = payload.State == "ON"
		o.online = true

		// Notify that the state has updated
		for len(o.updated) > 0 {
			<- o.updated
		}
		o.updated <- true

		// Notify google of the updated state
		id := o.Info.IEEEAdress
		s.ReportState(context.Background(), id, map[string]google.DeviceState{
			id: o.getState(),
		})

		if o.isOn {
			timer.Reset(length)
		} else {
			timer.Stop()
		}
	})

	o.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/get", o.Info.FriendlyName), 1, false, `{ "state": "" }`)

	return o
}

func (o* outlet) Sync() *google.Device {
	device := google.NewDevice(o.Info.IEEEAdress, google.TypeKettle)
	device.AddOnOffTrait(false, false)

	s := strings.Split(o.Info.FriendlyName, "/")
	room := ""
	name := s[0]
	if len(s) > 1 {
		room = s[0]
		name = s[1]
	}
	room = strings.Title(room)
	name = strings.Title(name)

	device.Name = google.DeviceName{
		DefaultNames: []string{
			"Kettle",
		},
		Name: name,
	}

	device.WillReportState = true
	if len(name) > 1 {
		device.RoomHint = room
	}

	device.DeviceInfo = google.DeviceInfo{
		Manufacturer: o.Info.Manufacturer,
		Model: o.Info.ModelID,
		SwVersion: o.Info.SoftwareBuildID,
	}

	o.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/get", o.Info.FriendlyName), 1, false, `{ "state": "" }`)

	return device
}

func (o *outlet) Query() google.DeviceState {
	// We just report out internal representation as it should always match the actual state
	state := o.getState()
	// No /get needed
	if o.online {
		state.Status = google.StatusSuccess
	} else {
		state.Status = google.StatusOffline
	}

	return state
}

func (o *outlet) Execute(execution google.Execution, updatedState *google.DeviceState) (string, bool) {
	errCode := ""

	switch execution.Name {
	case google.CommandOnOff:
		state := "OFF"
		if execution.OnOff.On {
			state = "ON"
		}

		// Clear the updated channel
		for len(o.updated) > 0 {
			<- o.updated
		}
		// Update the state
		o.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/set", o.Info.FriendlyName), 1, false, fmt.Sprintf(`{ "state": "%s" }`, state))

		// Start timeout timer
		timer := time.NewTimer(time.Second)

		// Wait for the update or timeout
		select {
			case <- o.updated:
				updatedState.RecordOnOff(o.isOn)

			case <- timer.C:
				// If we do not get a response in time mark the device as offline
				log.Println("Device did not respond, marking as offline")
				o.online = false
		}

	default:
		// @TODO Should probably move the error codes to a enum
		errCode = "actionNotAvailable"
		log.Printf("Command (%s) not supported\n", execution.Name)
	}

	return errCode, o.online
}
