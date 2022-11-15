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

func Connect() MQTT {
	host, ok := os.LookupEnv("MQTT_HOST")
	if !ok {
		host = "localhost"
	}
	port, ok := os.LookupEnv("MQTT_PORT")
	if !ok {
		port = "1883"
	}
	user, ok := os.LookupEnv("MQTT_USER")
	if !ok {
		user = "test"
	}
	pass, ok := os.LookupEnv("MQTT_PASS")
	if !ok {
		pass = "test"
	}
	clientID, ok := os.LookupEnv("MQTT_CLIENT_ID")
	if !ok {
		clientID = "automation"
	}

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", host, port))
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(user)
	opts.SetPassword(pass)

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
