package bootstrap

import "saya/config"

type Config struct {
	DatabaseURL string
	RabbitURL   string
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		DatabaseURL: r.URL("DATABASE_URL", "postgres", "postgresql"),
		RabbitURL:   r.URL("RABBIT_URL", "amqp", "amqps"),
	}

	return c, r.Err()
}
