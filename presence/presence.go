package presence

import (
	"automation/home"
	"automation/integration/hue"
	"automation/integration/ntfy"
	"encoding/json"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/kr/pretty"
)

type Presence struct {
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
	pretty.Logf("Presence updated: %v\n", p.devices)
	for _, value := range p.devices {
		if value {
			present = true
			break
		}
	}

	log.Printf("Setting overall presence: %t\n", present)

	if p.presence != present {
		p.presence = present

		payload, err := json.Marshal(Message{
			State:   present,
			Updated: time.Now().UnixMilli(),
		})
		if err != nil {
			log.Println(err)
		}

		token := client.Publish("automation/presence", 1, true, payload)
		if token.Wait() && token.Error() != nil {
			log.Println(token.Error())
		}
	}
}

func New(client paho.Client, hue *hue.Hue, ntfy *ntfy.Notify, home *home.Home) *Presence {
	p := &Presence{devices: make(map[string]bool), presence: false}

	if token := client.Subscribe("automation/presence/+", 1, p.devicePresenceHandler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

	return p
}

func (p *Presence) Delete(client paho.Client) {
	if token := client.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}
