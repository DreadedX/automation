package mqtt

import (
	"fmt"
	"os"

	"github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	client   mqtt.Client
}

// This is the default message handler, it just prints out the topic and message
var defaultHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func Connect(config Config) MQTT {
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", config.Host, config.Port))
	opts.SetClientID(config.ClientID)
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetOrderMatters(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	m := MQTT{client: client}

	return m
}

func (m *MQTT) Disconnect() {
	if token := m.client.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	m.client.Disconnect(250)
}

func (m *MQTT) AddHandler(topic string, handler func(client mqtt.Client, msg mqtt.Message)) {
	if token := m.client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (m *MQTT) Publish(topic string, qos byte, retained bool, payload interface{}) {
	if token := m.client.Publish(topic, qos, retained, payload); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		// Do not exit here as it might break during production, just log the error
		// os.Exit(1)
	}
}
