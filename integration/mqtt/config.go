package mqtt

type Config struct {
	Host     string `yaml:"host" envconfig:"MQTT_HOST"`
	Port     string `yaml:"port" envconfig:"MQTT_PORT"`
	Username string `yaml:"username" envconfig:"MQTT_USERNAME"`
	Password string `yaml:"password" envconfig:"MQTT_PASSWORD"`
	ClientID string `yaml:"client_id" envconfig:"MQTT_CLIENT_ID"`
}

