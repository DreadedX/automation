package device

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

type kettle struct {
	Info DeviceInfo
	m *mqtt.MQTT
	updated chan bool

	isOn bool
	online bool
}

func (k *kettle) getState() google.DeviceState {
	return google.NewDeviceState(k.online).RecordOnOff(k.isOn)
}

func NewKettle(info DeviceInfo, m *mqtt.MQTT, s *google.Service) *kettle {
	k := &kettle{Info: info, m: m, updated: make(chan bool, 1)}

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

	k.m.AddHandler(fmt.Sprintf("zigbee2mqtt/%s", k.Info.FriendlyName), func (_ paho.Client, msg paho.Message)  {
		var payload struct {
			State string `json:"state"`
		}
		json.Unmarshal(msg.Payload(), &payload)

		// Update the internal state
		k.isOn = payload.State == "ON"
		k.online = true

		// Notify that the state has updated
		for len(k.updated) > 0 {
			<- k.updated
		}
		k.updated <- true

		// Notify google of the updated state
		id := k.GetID()
		s.ReportState(context.Background(), id, map[string]google.DeviceState{
			id: k.getState(),
		})

		if k.isOn {
			timer.Reset(length)
		} else {
			timer.Stop()
		}
	})

	k.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/get", k.Info.FriendlyName), 1, false, `{ "state": "" }`)

	return k
}

func (k *kettle) Sync() *google.Device {
	device := google.NewDevice(k.GetID(), google.TypeKettle)
	device.AddOnOffTrait(false, false)

	s := strings.Split(k.Info.FriendlyName, "/")
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
		Manufacturer: k.Info.Manufacturer,
		Model: k.Info.ModelID,
		SwVersion: k.Info.SoftwareBuildID,
	}

	k.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/get", k.Info.FriendlyName), 1, false, `{ "state": "" }`)

	return device
}

func (k *kettle) Query() google.DeviceState {
	// We just report out internal representation as it should always match the actual state
	state := k.getState()
	// No /get needed
	if k.online {
		state.Status = google.StatusSuccess
	} else {
		state.Status = google.StatusOffline
	}

	return state
}

func (k *kettle) Execute(execution google.Execution, updatedState *google.DeviceState) (string, bool) {
	errCode := ""

	switch execution.Name {
	case google.CommandOnOff:
		state := "OFF"
		if execution.OnOff.On {
			state = "ON"
		}

		// Clear the updated channel
		for len(k.updated) > 0 {
			<- k.updated
		}
		// Update the state
		k.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/set", k.Info.FriendlyName), 1, false, fmt.Sprintf(`{ "state": "%s" }`, state))

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

func (k *kettle) GetID() string {
	return k.Info.IEEEAdress
}

func (k *kettle) TurnOff() {
	k.m.Publish(fmt.Sprintf("zigbee2mqtt/%s/set", k.Info.FriendlyName), 1, false, fmt.Sprintf(`{ "state": "OFF" }`))
}
