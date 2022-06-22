package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amimof/huego"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/kelvins/sunrisesunset"
)

// @TODO Make this environment variables
// SETTINGS
// Group id of the lights in the living room
const groupId = 1
// The color temperature to use for the lights
// @TODO Not sure how this is calulcated
const Temperature uint16 = 366

// All the different message types
type Type string
const (
	Beacon        Type = "beacon"
	Card               = "card"
	Cmd                = "cmd"
	Configuration      = "configuration"
	Encrypted          = "encrypted"
	Location           = "location"
	Lwt                = "lwt"
	Steps              = "steps"
	Transition         = "transition"
	Waypoint           = "waypoint"
	Waypoints          = "waypoints"
)

// Struct for parsing message type
type Identifier struct {
	Type Type `json:"_type"`
}

// Struct with all the data from location messages
type LocationData struct {
	Longitude float32  `json:"lon"`
	Latitude  float32  `json:"lat"`
	Altitude  int      `json:"alt"`
	InRegions []string `json:"inregions"`
}

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

// Turn off all the lights attached to the bridge
func allLightsOff(bridge *huego.Bridge) {
	lights, _ := bridge.GetLights()
	for _, l := range lights {
		l.Off()
	}
}

// This is the default message handler, it just prints out the topic and message
var defaultHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

// Handler got owntrack/+/+
func locationHandler(home chan bool) func(MQTT.Client, MQTT.Message) {
	return func(client MQTT.Client, msg MQTT.Message) {
		// Check the type of the MQTT message
		var identifier Identifier
		json.Unmarshal(msg.Payload(), &identifier)
		if identifier.Type != Location {
			return
		}

		// Marshall all the location data
		var location LocationData
		json.Unmarshal(msg.Payload(), &location)

		fmt.Println(location)
		temp := false
		for _, region := range location.InRegions {
			if region == "home" {
				temp = true
			}
		}
		fmt.Println(temp)

		home <- temp
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
	login, _ := os.LookupEnv("HUE_BRIDGE")

	halt := make(chan os.Signal, 1)
	signal.Notify(halt, os.Interrupt, syscall.SIGTERM)

	// bridge, _ := huego.Discover()
	// bridge = bridge.Login(login)
	// @TODO Let's hope the IP does not change, should probably set a static IP
	bridge := huego.New("10.0.0.146", login)
	if bridge == nil {
		panic("Bridge is nil")
	}

	opts := MQTT.NewClientOptions().AddBroker(fmt.Sprintf("%s:%s", host, port))
	opts.SetClientID("automation")
	opts.SetDefaultPublishHandler(defaultHandler)
	opts.SetUsername(user)
	opts.SetPassword(pass)

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	home := make(chan bool, 1)
	if token := c.Subscribe("owntracks/mqtt/apollo", 0, locationHandler(home)); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Setup initial states
	isHome := false

	sunrise, sunset := getNextSunriseSunset()
	sunriseTimer := time.NewTimer(sunrise.Sub(time.Now()))
	sunsetTimer := time.NewTimer(sunset.Sub(time.Now()))

	// Create the ticker, but stop it
	ticker := time.NewTicker(time.Second)
	ticker.Stop()

	var brightness uint8 = 1

	// Event loop
events:
	for {
		select {
		case temp := <-home:
			// Only do stuff if the state changes
			if temp == isHome {
				break
			}
			isHome = temp

			if isHome {
				fmt.Println("Coming home")
				if !isDay() {
					fmt.Println("\tTurning on lights in the living room")
					livingRoom, _ := bridge.GetGroup(groupId)
					livingRoom.Bri(0xff)
					livingRoom.Ct(Temperature)
				}
			} else {
				// Stop the ticker in case it is running
				ticker.Stop()

				fmt.Println("Leaving home")
				allLightsOff(bridge)
				break
			}

		case <-sunriseTimer.C:
			fmt.Println("Sun is rising, turning off all lights")
			allLightsOff(bridge)

			// Set new timer
			sunrise, _ := getNextSunriseSunset()
			sunriseTimer.Reset(sunrise.Sub(time.Now()))

		case <-sunsetTimer.C:
			fmt.Println("Sun is setting")
			if isHome {
				fmt.Println("\tGradually turning on lights in the living room")
				// Start the ticker to gradually turn on the living room lights
				ticker.Reset(1200 * time.Millisecond)

				livingRoom, _ := bridge.GetGroup(groupId)

				fmt.Println("DEBUG STUFG")
				fmt.Println(livingRoom.IsOn())
				fmt.Println(livingRoom.State.On)
				fmt.Println(livingRoom.State.Bri)
				fmt.Println(livingRoom.State.Ct)
				fmt.Println(brightness)

				if (!livingRoom.IsOn() || livingRoom.State.Bri < brightness) {
					fmt.Println("Setting brightness:", brightness)
					livingRoom.Bri(brightness)
					livingRoom.Ct(Temperature)
				}
			}

			// Set new timer
			_, sunset := getNextSunriseSunset()
			sunsetTimer.Reset(sunset.Sub(time.Now()))

		case <-ticker.C:
			brightness++
			livingRoom, _ := bridge.GetGroup(groupId)
			if (!livingRoom.IsOn() || livingRoom.State.Bri < brightness) {
				fmt.Println("Setting brightness:", brightness)
				livingRoom.Bri(brightness)
				livingRoom.Ct(Temperature)
			}

			if brightness == 0xff {
				fmt.Println("Lights are now on, stopping ticker")
				ticker.Stop()
				brightness = 1
			}

		case <-halt:
			break events
		}
	}

	// Cleanup
	if token := c.Unsubscribe("owntracks/mqtt/apollo"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}
