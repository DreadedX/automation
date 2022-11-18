package mqtt

import (
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// This is the default message handler, it just prints out the topic and message
var defaultHandler paho.MessageHandler = func(client paho.Client, msg paho.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func New(config Config) paho.Client {
	opts := paho.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", config.Host, config.Port))
	opts.SetClientID(config.ClientID)
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetOrderMatters(false)

	client := paho.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return client
}

func Delete(m paho.Client) {
	if token := m.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	m.Disconnect(250)
}
