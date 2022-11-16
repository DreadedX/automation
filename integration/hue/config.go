package hue

type Config struct {
	Token string `yaml:"token" envconfig:"HUE_TOKEN"`
	IP    string `yaml:"ip" envconfig:"HUE_IP"`
}
