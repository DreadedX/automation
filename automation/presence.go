package automation

import (
	"automation/device"
	"automation/home"
	"automation/integration/hue"
	"automation/integration/ntfy"
	"automation/presence"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func presenceAutomation(client paho.Client, hue *hue.Hue, notify *ntfy.Notify, home *home.Home) {
	on(client, "automation/presence", func(message presence.Message) {
		log.Printf("Presence changed: %t\n", message.State)

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
	})
}
