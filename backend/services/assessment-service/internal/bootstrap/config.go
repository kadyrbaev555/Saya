package bootstrap

import "saya/config"

type Config struct {
	DatabaseURL      string
	RabbitURL        string
	SurveyAddress    string
	CentrifugoURL    string
	CentrifugoAPIKey string
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		DatabaseURL:      r.URL("DATABASE_URL", "postgres", "postgresql"),
		RabbitURL:        r.URL("RABBIT_URL", "amqp", "amqps"),
		SurveyAddress:    r.Address("SURVEY_ADDRESS", "survey:9000"),
		CentrifugoURL:    r.URL("CENTRIFUGO_URL", "http", "https"),
		CentrifugoAPIKey: r.Required("CENTRIFUGO_API_KEY"),
	}

	return c, r.Err()
}
