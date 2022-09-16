package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

// This is the default message handler, it just prints out the topic and message
var defaultHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

// Handler got automation/presence/+
func presenceHandler(presence chan bool) func(MQTT.Client, MQTT.Message) {
	devices := make(map[string]bool)
	var current *bool

	return func(client MQTT.Client, msg MQTT.Message) {
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
				break;
			}
		}

		if current == nil || *current != present {
			current = &present
			presence <- present
		}

	}
}

type Hue struct {
	ip string
	login string
}

func (hue *Hue) updateFlag(id int, value bool) {
	url := fmt.Sprintf("http://%s/api/%s/sensors/%d/state", hue.ip, hue.login, id)

	var data []byte
	if value {
		data = []byte(`{ "flag": true }`)
	} else {
		data = []byte(`{ "flag": false }`)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
}

type ntfy struct{
	topic string
}

func (ntfy *ntfy) notifyPresence(home bool) {
	// @TODO Maybe add list the devices that are home currently?
	var description string
	var actions string
	if home {
		description = "Home"
		actions = "broadcast, Set as away, extras.cmd=presence, extras.state=0, clear=true"
	} else {
		description = "Away"
		actions = "broadcast, Set as home, extras.cmd=presence, extras.state=1, clear=true"
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://ntfy.sh/%s", ntfy.topic), strings.NewReader(description))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Title", "Presence")
	req.Header.Set("Tags", "house")
	req.Header.Set("Actions", actions)
	req.Header.Set("Priority", "1")

	http.DefaultClient.Do(req)
}

func connectToHue() Hue {
	login, _ := os.LookupEnv("HUE_BRIDGE")

	// @TODO Discover the bridge here
	hue := Hue{ip: "10.0.0.146", login: login}

	// @TODO Make sure we actually are connected here

	return hue
}

func connectMQTT() MQTT.Client {
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

	opts := MQTT.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", host, port))
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(user)
	opts.SetPassword(pass)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return client
}

func connectNtfy() ntfy {
	topic, _ := os.LookupEnv("NTFY_TOPIC")
	ntfy := ntfy{topic}

	// @TODO Make sure the topic is valid?

	return ntfy
}

func main() {
	_ = godotenv.Load()

	// Signals
	halt := make(chan os.Signal, 1)
	signal.Notify(halt, os.Interrupt, syscall.SIGTERM)

	// MQTT
	client := connectMQTT()
	presence := make(chan bool, 1)
	if token := client.Subscribe("automation/presence/+", 0, presenceHandler(presence)); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	// Hue
	hue := connectToHue()

	// ntfy.sh
	ntfy := connectNtfy()

	// Event loop
	fmt.Println("Starting event loop")
events:
	for {
		select {
		case present := <-presence:
			fmt.Printf("Present: %t", present)
			hue.updateFlag(41, present)
			ntfy.notifyPresence(present)

		case <-halt:
			break events
		}
	}

	// Cleanup
	if token := client.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	client.Disconnect(250)
}
