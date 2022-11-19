package zigbee

import (
	"automation/device"
	"automation/home"
	"context"
	"encoding/json"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func DevicesHandler(client paho.Client, home *home.Home) {
	var handler paho.MessageHandler = func(client paho.Client, msg paho.Message) {
		var devices []Info
		json.Unmarshal(msg.Payload(), &devices)

		for name, d := range device.GetDevices[Device](&home.Devices) {
			d.Delete()
			// Delete all zigbee devices from the device list
			delete(home.Devices, name)
		}

		for _, d := range devices {
			switch d.Description {
			case "Kettle":
				kettle := NewKettle(d, client, home.Service)
				home.AddDevice(kettle)
			}
		}

		// Send sync request
		home.Service.RequestSync(context.Background(), home.Username)
	}

	if token := client.Subscribe("zigbee2mqtt/bridge/devices", 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}
