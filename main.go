package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/r3labs/sse/v2"
)

// This is the default message handler, it just prints out the topic and message
var defaultHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

type DeviceStatus struct {
	Name    string
	Present bool
}

// Handler got automation/presence/+
func presenceHandler(status chan DeviceStatus) func(MQTT.Client, MQTT.Message) {
	return func(client MQTT.Client, msg MQTT.Message) {
		device := strings.Split(msg.Topic(), "/")[2]
		value, err := strconv.Atoi(string(msg.Payload()))
		if err != nil {
			panic(err)
		}

		fmt.Println("presenceHandler", device, value)

		status <- DeviceStatus{Name: device, Present: value == 1}
	}
}

type Hue struct {
	ip    string
	login string
}

func (hue *Hue) listenForEvents(events chan *sse.Event) {
	client := sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", hue.ip))
	client.Headers["hue-application-key"] = hue.login
	client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	err := client.SubscribeChanRaw(events)
	if err != nil {
		panic(err)
	}
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

type lastEvent struct {
	LastEvent string `json:"last_event"`
}

type owner struct {
	Rid   string `json:"rid"`
	Rtype string `json:"rtype"`
}

type Button struct {
	Button lastEvent `json:"button"`
	Id     string    `json:"id"`
	IdV1   string    `json:"id_v1"`
	Owner  owner     `json:"owner"`
	Type   string    `json:"type"`
}

type on struct {
	On bool `json:"on"`
}

type typeInfo struct {
	Type string `json:"type"`
}

type Event struct {
	CreationTime time.Time       `json:"creationtime"`
	Data         []json.RawMessage `json:"data"`
	ID           string          `json:"id"`
	Type         string          `json:"type"`
}

func main() {
	_ = godotenv.Load()

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
	login, _ := os.LookupEnv("HUE_BRIDGE")

	halt := make(chan os.Signal, 1)
	signal.Notify(halt, os.Interrupt, syscall.SIGTERM)

	// @TODO Discover the bridge here
	hue := Hue{ip: "10.0.0.146", login: login}
	events := make(chan *sse.Event)
	hue.listenForEvents(events)

	opts := MQTT.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", host, port))
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(user)
	opts.SetPassword(pass)

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	status := make(chan DeviceStatus, 1)
	if token := c.Subscribe("automation/presence/+", 0, presenceHandler(status)); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	devices := make(map[string]bool)
	isHome := false

	fmt.Println("Starting event loop")

	// Event loop
events:
	for {
		select {
		case state := <-status:
			// Update the device state
			devices[state.Name] = state.Present

			// Check if there is any device home
			temp := false
			for key, value := range devices {
				fmt.Println(key, value)
				if value {
					temp = true
					break
				}
			}

			// Only do stuff if the state changes
			if temp == isHome {
				break
			}
			isHome = temp

			if isHome {
				fmt.Println("Coming home")
				hue.updateFlag(41, true)
			} else {
				fmt.Println("Leaving home")
				hue.updateFlag(41, false)
			}

			fmt.Println("Done")

		case message := <-events:
			events := []Event{}
			json.Unmarshal(message.Data, &events)

			for _, event := range events {
				if event.Type == "update" {
					for _, data := range event.Data {
						var typeInfo typeInfo
						json.Unmarshal(data, &typeInfo)

						switch typeInfo.Type {
						case "button":
							fmt.Println("Button")
							var button Button
							json.Unmarshal(data, &button)
							fmt.Println(button)
						}
					}
				}
			}

		case <-halt:
			break events
		}
	}

	// Cleanup
	if token := c.Unsubscribe("automation/presence/+"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}
