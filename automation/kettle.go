package automation

import (
	"automation/device"
	"automation/home"
	"automation/integration/zigbee"
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func kettleAutomation(client paho.Client, prefix string, home *home.Home) {
	const name = "kitchen/kettle"
	const length = 5 * time.Minute

	timer := time.NewTimer(length)

	on(client, fmt.Sprintf("%s/%s", prefix, name), func(message zigbee.OnOffState) {
		if message.State {
			timer.Reset(length)
		} else {
			timer.Stop()
		}
	})

	go func() {
		for {
			<-timer.C
			log.Println("Turning kettle automatically off")
			kettle, err := device.GetDevice[device.OnOff](&home.Devices, name)
			if err != nil {
				log.Println(err)
				break
			}

			kettle.SetOnOff(false)
		}
	}()
}
