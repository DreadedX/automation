package main

import (
	"bytes"
	"crypto/tls"
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
	"github.com/kelvins/sunrisesunset"
)

// Get the time of the next sunrise and sunset
func getNextSunriseSunset() (time.Time, time.Time) {
	p := sunrisesunset.Parameters{
		Latitude:  51.9808334,
		Longitude: 4.347818,
		Date:      time.Now(),
	}

	sunrise, sunset, err := p.GetSunriseSunset()
	if err != nil {
		panic(err)
	}
	sunset = sunset.Add(-time.Minute*30)

	p2 := sunrisesunset.Parameters{
		Latitude:  51.9808334,
		Longitude: 4.347818,
		Date:      time.Now().Add(time.Hour * 24),
	}
	sunrise2, sunset2, err := p2.GetSunriseSunset()
	if err != nil {
		panic(err)
	}
	sunset2 = sunset2.Add(-time.Minute*30)

	now := time.Now()
	if now.After(sunrise) {
		sunrise = sunrise2
	}

	if now.After(sunset) {
		sunset = sunset2
	}

	return sunrise, sunset
}

// Check if it is currently daytime
func isDay() bool {
	p := sunrisesunset.Parameters{
		Latitude:  51.9808334,
		Longitude: 4.347818,
		Date:      time.Now(),
	}

	sunrise, sunset, err := p.GetSunriseSunset()
	if err != nil {
		panic(err)
	}
	sunset = sunset.Add(-time.Minute*30)

	return time.Now().After(sunrise) && time.Now().Before(sunset)
}

// This is the default message handler, it just prints out the topic and message
var defaultHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

type DeviceStatus struct {
	Name string
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
	ip string
	login string
}

func (hue *Hue) putRequest(resource string, data string) {
	url := fmt.Sprintf("https://%s/clip/v2/resource/%s", hue.ip, resource)

	client := &http.Client{}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("hue-application-key", hue.login)

	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
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

	sunrise, sunset := getNextSunriseSunset()
	sunriseTimer := time.NewTimer(sunrise.Sub(time.Now()))
	sunsetTimer := time.NewTimer(sunset.Sub(time.Now()))

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
					break;
				}
			}

			// Only do stuff if the state changes
			if temp == isHome {
				break
			}
			isHome = temp

			if isHome {
				fmt.Println("Coming home")
				if !isDay() {
					fmt.Println("\tTurning on lights in the living room")
					hue.putRequest("scene/1847ec79-3459-4d79-ae73-803a0c6e7ac2", `{"recall": { "action": "active", "status": "active"}}`)
				}
			} else {
				fmt.Println("Leaving home")
				hue.putRequest("grouped_light/91c400ed-7eda-4b5c-ac3f-bfff226188d7", `{"on": { "on": false}}`)
				break
			}

			fmt.Println("Done")

		case <-sunriseTimer.C:
			fmt.Println("Sun is rising, turning off all lights")
			hue.putRequest("grouped_light/91c400ed-7eda-4b5c-ac3f-bfff226188d7", `{"on": { "on": false}}`)

			// Set new timer
			sunrise, _ := getNextSunriseSunset()
			sunriseTimer.Reset(sunrise.Sub(time.Now()))

		case <-sunsetTimer.C:
			fmt.Println("Sun is setting")
			if isHome {
				fmt.Println("\tGradually turning on lights in the living room")
				hue.putRequest("scene/1847ec79-3459-4d79-ae73-803a0c6e7ac2", `{"recall": { "action": "active", "status": "active", "duration": 300000}}`)
			}

			// Set new timer
			_, sunset := getNextSunriseSunset()
			sunsetTimer.Reset(sunset.Sub(time.Now()))

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
