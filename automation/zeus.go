package automation

import (
	"automation/device"
	"automation/home"
	"fmt"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func zeusAutomation(client paho.Client, home *home.Home) {
	const name = "living_room/zeus"
	on(client, fmt.Sprintf("automation/appliance/%s", name), func(message struct{Activate bool `json:"activate"`}) {
		computer, err := device.GetDevice[device.Activate](&home.Devices, name)
		if err != nil {
			log.Println(err)
			return
		}

		computer.Activate(message.Activate)
	})
}
