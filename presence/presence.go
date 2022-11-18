package presence

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kr/pretty"
)

type Presence struct {
	devices  map[string]bool
	current  *bool
}

type Message struct {
	State   bool  `json:"state"`
	Updated int64 `json:"updated"`
}

func New() Presence {
	return Presence{devices: make(map[string]bool), current: nil}
}

// Handler got automation/presence/+
func (p *Presence) PresenceHandler(client mqtt.Client, msg mqtt.Message) {
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

	if p.current == nil || *p.current != present {
		p.current = &present

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
