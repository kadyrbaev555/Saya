package bootstrap

import "saya/config"

type Config struct {
	AssessmentAddress string
	RabbitURL         string
	LLMProvider       string
	LLMEndpoint       string
	LLMAPIKey         string
	GeminiAPIKey      string
	GeminiModel       string
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		AssessmentAddress: r.Address("ASSESSMENT_ADDRESS", "assessment:9000"),
		RabbitURL:         r.URL("RABBIT_URL", "amqp", "amqps"),
		LLMProvider:       r.String("LLM_PROVIDER", "mock"),
		LLMEndpoint:       r.String("LLM_ENDPOINT", ""),
		LLMAPIKey:         r.String("LLM_API_KEY", ""),
		GeminiAPIKey:      r.String("GEMINI_API_KEY", ""),
		GeminiModel:       r.String("GEMINI_MODEL", ""),
	}
	switch c.LLMProvider {
	case "mock":
	case "http":
		c.LLMEndpoint = r.URL("LLM_ENDPOINT", "https")
		c.LLMAPIKey = r.Required("LLM_API_KEY")
	case "gemini":
		c.GeminiAPIKey = r.Required("GEMINI_API_KEY")
		c.GeminiModel = r.Required("GEMINI_MODEL")
	default:
		r.Invalid("LLM_PROVIDER")
	}
	return c, r.Err()
}
