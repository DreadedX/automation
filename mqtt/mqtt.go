package mqtt

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	Presence chan bool
	client   mqtt.Client
}

// This is the default message handler, it just prints out the topic and message
var defaultHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

// Handler got automation/presence/+
func presenceHandler(presence chan bool) func(mqtt.Client, mqtt.Message) {
	devices := make(map[string]bool)
	var current *bool

	return func(client mqtt.Client, msg mqtt.Message) {
		name := strings.Split(msg.Topic(), "/")[2]
		if len(msg.Payload()) == 0 {
			// @TODO What happens if we delete a device that does not exist
			delete(devices, name)
		} else {
			value, err := strconv.Atoi(string(msg.Payload()))
			if err != nil {
				panic(err)
			}

			devices[name] = value == 1
		}

		present := false
		fmt.Println(devices)
		for _, value := range devices {
			if value {
				present = true
				break
			}
		}

		if current == nil || *current != present {
			current = &present
			presence <- present
		}

	}
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

	m := MQTT{client: client, Presence: make(chan bool)}

	if token := client.Subscribe("automation/presence/+", 0, presenceHandler(m.Presence)); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	return m
}

func (m *MQTT) Disconnect() {
	if token := m.client.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	m.client.Disconnect(250)
}
