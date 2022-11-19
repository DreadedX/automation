package automation

import (
	"automation/device"
	"automation/home"
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func kettleAutomation(client paho.Client, home *home.Home) {
	const name = "kitchen/kettle"
	const length = 5 * time.Minute

	timer := time.NewTimer(length)

	var handler paho.MessageHandler = func(c paho.Client, m paho.Message) {
		kettle, err := device.GetDevice[device.OnOff](&home.Devices, name)
		if err != nil {
			log.Println(err)
			return
		}

		if kettle.GetOnOff() {
			timer.Reset(length)
		} else {
			timer.Stop()
		}
	}

	if token := client.Subscribe(fmt.Sprintf("zigbee2mqtt/%s", name), 1, handler); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}

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
