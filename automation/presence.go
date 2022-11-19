package automation

import (
	"automation/device"
	"automation/home"
	"automation/integration/hue"
	"automation/integration/ntfy"
	"automation/presence"
	"encoding/json"
	"fmt"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func presenceAutomation(client paho.Client, hue *hue.Hue, notify *ntfy.Notify, home *home.Home) {
	var handler paho.MessageHandler = func(client paho.Client, msg paho.Message) {
		if len(msg.Payload()) == 0 {
			// In this case we clear the persistent message
			return
		}
		var message presence.Message
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Printf("Presence: %t\n", message.State)

		// Set presence on the hue bridge
		hue.SetFlag(41, message.State)

		if !message.State {
			log.Println("Turn off all the devices")

			// Turn off all devices
			// @TODO Maybe allow for exceptions, could be a list in the config that we check against?
			for _, dev := range home.Devices {
				switch d := dev.(type) {
				case device.OnOff:
					d.SetOnOff(false)
				}

			}

			// @TODO Turn off nest thermostat
		} else {
			// @TODO Turn on the nest thermostat again
		}

		// Notify users of presence update
		notify.Presence(message.State)
	}

	if token := client.Subscribe("automation/presence", 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}
