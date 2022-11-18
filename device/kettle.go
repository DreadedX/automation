package device

import (
	"automation/integration/google"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type kettle struct {
	info DeviceInfo
	client paho.Client
	updated chan bool

	timerLength time.Duration
	timer *time.Timer
	stop chan interface{}

	isOn bool
	online bool
}

func (k *kettle) getState() google.DeviceState {
	return google.NewDeviceState(k.online).RecordOnOff(k.isOn)
}

func (k *kettle) stateHandler(client paho.Client, msg paho.Message) {
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
	// @TODO Fix this
	// id := k.GetID()
	// s.ReportState(context.Background(), id, map[string]google.DeviceState{
	// 	id: k.getState(),
	// })

	if k.isOn {
		k.timer.Reset(k.timerLength)
	} else {
		k.timer.Stop()
	}
}

func (k *kettle) timerFunc() {
	for {
		select {
		case <- k.timer.C:
			log.Println("Turning kettle automatically off")
			if token := k.client.Publish(fmt.Sprintf("zigbee2mqtt/%s/set", k.info.FriendlyName), 1, false, `{"state": "OFF"}`); token.Wait() && token.Error() != nil {
				log.Println(token.Error())
			}

		case <- k.stop:
			return
		}
	}
}

func (k *kettle) Delete() {
	// The the timer function that it needs to stop
	k.stop <- struct{}{}
}

func NewKettle(info DeviceInfo, client paho.Client, s *google.Service) *kettle {
	k := &kettle{info: info, client: client, updated: make(chan bool, 1), timerLength: 5 * time.Minute, stop: make(chan interface{})}
	k.timer = time.NewTimer(k.timerLength)
	k.timer.Stop()

	// Start function 
	go k.timerFunc()

	if token := k.client.Subscribe(fmt.Sprintf("zigbee2mqtt/%s", k.info.FriendlyName), 1, k.stateHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	return k
}

func (k *kettle) Sync() *google.Device {
	device := google.NewDevice(k.GetID(), google.TypeKettle)
	device.AddOnOffTrait(false, false)

	s := strings.Split(k.info.FriendlyName, "/")
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

	// @TODO Fix reporting
	// device.WillReportState = true
	device.WillReportState = true
	if len(name) > 1 {
		device.RoomHint = room
	}

	device.DeviceInfo = google.DeviceInfo{
		Manufacturer: k.info.Manufacturer,
		Model: k.info.ModelID,
		SwVersion: k.info.SoftwareBuildID,
	}

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

		// Clear the updated channel
		for len(k.updated) > 0 {
			<- k.updated
		}

		k.SetState(execution.OnOff.On)

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
	return k.info.IEEEAdress
}

func (k *kettle) GetName() string {
	return k.info.FriendlyName
}

func (k *kettle) GetDeviceInfo() DeviceInfo {
	return k.info
}

func (k *kettle) SetState(state bool) {
	msg := "OFF"
	if state {
		msg = "ON"
	}

	if token := k.client.Publish(fmt.Sprintf("zigbee2mqtt/%s/set", k.info.FriendlyName), 1, false, fmt.Sprintf(`{ "state": "%s" }`, msg)); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

