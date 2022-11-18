package presence

import (
	"automation/connect"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/kr/pretty"
)

type Presence struct {
	connect *connect.Connect
	devices  map[string]bool
	presence  bool
}

type Message struct {
	State   bool  `json:"state"`
	Updated int64 `json:"updated"`
}

func (p *Presence) devicePresenceHandler(client paho.Client, msg paho.Message) {
	name := strings.Split(msg.Topic(), "/")[2]

	if len(msg.Payload()) == 0 {
		delete(p.devices, name)
	} else {
		var message Message
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			log.Println(err)
			return
		}

		p.devices[name] = message.State
	}

	present := false
	pretty.Println(p.devices)
	for _, value := range p.devices {
		if value {
			present = true
			break
		}
	}

	log.Println(present)

	if p.presence != present {
		p.presence = present

		msg, err := json.Marshal(Message{
			State:   present,
			Updated: time.Now().UnixMilli(),
		})

		if err != nil {
			log.Println(err)
		}

		token := client.Publish("automation/presence", 1, true, msg)
		if token.Wait() && token.Error() != nil {
			log.Println(token.Error())
		}
	}
}

func (p *Presence) overallPresenceHandler(client paho.Client, msg paho.Message) {
	if len(msg.Payload()) == 0 {
		// In this case we clear the persistent message
		return
	}
	var message Message
	err := json.Unmarshal(msg.Payload(), &message)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Printf("Presence: %t\n", message.State)
	// Notify users of presence update
	p.connect.Notify.Presence(p.presence)

	// Set presence on the hue bridge
	p.connect.Hue.SetFlag(41, message.State)

	if !message.State {
		log.Println("Turn off all the devices")
		// // Turn off all the devices that we manage ourselves
		// provider.TurnAllOff()

		// // Turn off all devices
		// // @TODO Maybe allow for exceptions, could be a list in the config that we check against?
		// for _, device := range devices {
		// 	switch d := device.(type) {
		// 	case kasa.Kasa:
		// 		d.SetState(false)

		// 	}
		// }

		// @TODO Turn off nest thermostat
	} else {
		// @TODO Turn on the nest thermostat again
	}
}

func New(connect *connect.Connect) *Presence {
	p := &Presence{connect: connect, devices: make(map[string]bool), presence: false}

	if token := connect.Client.Subscribe("automation/presence", 1, p.overallPresenceHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	if token := connect.Client.Subscribe("automation/presence/+", 1, p.devicePresenceHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	return p
}

func (p *Presence) Delete() {
	if token := p.connect.Client.Unsubscribe("automation/presence"); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	if token := p.connect.Client.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}
