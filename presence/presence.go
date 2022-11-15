package presence

import (
	"fmt"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Presence struct {
	Presence chan bool
	devices map[string]bool
	current *bool
}

func New() Presence {
	return Presence{Presence: make(chan bool), devices: make(map[string]bool), current: nil}
}

// Handler got automation/presence/+
func (p *Presence) PresenceHandler(client mqtt.Client, msg mqtt.Message) {
	name := strings.Split(msg.Topic(), "/")[2]
	if len(msg.Payload()) == 0 {
		// @TODO What happens if we delete a device that does not exist
		delete(p.devices, name)
	} else {
		value, err := strconv.Atoi(string(msg.Payload()))
		if err != nil {
			panic(err)
		}

		p.devices[name] = value == 1
	}

	present := false
	fmt.Println(p.devices)
	for _, value := range p.devices {
		if value {
			present = true
			break
		}
	}

	if p.current == nil || *p.current != present {
		p.current = &present
		p.Presence <- present
	}
}

