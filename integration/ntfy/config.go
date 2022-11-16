package ntfy

type Config struct {
	Presence string `yaml:"presence" envconfig:"NTFY_PRESENCE"`
}
