package config

import (
	"automation/device"
	"encoding/base64"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type config struct {
	Hue struct {
		Token string `yaml:"token" envconfig:"HUE_TOKEN"`
		IP    string `yaml:"ip" envconfig:"HUE_IP"`
	} `yaml:"hue"`

	Ntfy struct {
		Topic string `yaml:"topic" envconfig:"NTFY_TOPIC"`
	} `yaml:"ntfy"`

	MQTT struct {
		Host     string `yaml:"host" envconfig:"MQTT_HOST"`
		Port     int    `yaml:"port" envconfig:"MQTT_PORT"`
		Username string `yaml:"username" envconfig:"MQTT_USERNAME"`
		Password string `yaml:"password" envconfig:"MQTT_PASSWORD"`
		ClientID string `yaml:"client_id" envconfig:"MQTT_CLIENT_ID"`
	} `yaml:"mqtt"`

	Zigbee struct {
		MQTTPrefix string `yaml:"prefix" envconfig:"ZIGBEE2MQTT_PREFIX"`
	}

	Kasa struct {
		Outlets map[device.InternalName]string `yaml:"outlets"`
	} `yaml:"kasa"`

	Computer map[device.InternalName]struct {
		MACAddress string `yaml:"mac"`
		Url        string `yaml:"url"`
	} `yaml:"computers"`

	Google struct {
		Username string `yaml:"username" envconfig:"GOOGLE_USERNAME"`
		Credentials Credentials `yaml:"credentials" envconfig:"GOOGLE_CREDENTIALS"`
	} `yaml:"google"`
}

type Credentials []byte

func (c *Credentials) Decode(value string) error {
	b, err := base64.StdEncoding.DecodeString(value)
	*c = b

	return err
}

func Get() config {
	// First load the config from the yaml file
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalln("Failed to open config file", err)
	}
	defer f.Close()

	var cfg config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalln("Failed to parse config file", err)
	}

	// Then load values from environment
	// This can be used to either override the config or pass in secrets
	err = envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalln("Failed to parse environmet config", err)
	}

	return cfg
}
