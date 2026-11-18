package bootstrap

import "saya/config"

type Config struct {
	SMTPAddress, SMTPFrom, SMTPUser, SMTPPassword string
	DatabaseURL                                   string
	JWTSeed                                       string
	JWTKid                                        string
	CentrifugoTokenSecret                         string
	ArgonMemory                                   uint32
	ArgonIterations                               uint32
	ArgonThreads                                  uint8
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		SMTPAddress:           r.String("SMTP_ADDRESS", ""),
		SMTPFrom:              r.String("SMTP_FROM", ""),
		SMTPUser:              r.String("SMTP_USERNAME", ""),
		SMTPPassword:          r.String("SMTP_PASSWORD", ""),
		DatabaseURL:           r.URL("DATABASE_URL", "postgres", "postgresql"),
		JWTSeed:               r.Key("JWT_SEED"),
		JWTKid:                r.String("JWT_KID", "local-1"),
		CentrifugoTokenSecret: r.Required("CENTRIFUGO_TOKEN_SECRET"),
		ArgonMemory:           uint32(r.Uint("ARGON_MEMORY_KIB", "65536", 65536, 262144)),
		ArgonIterations:       uint32(r.Uint("ARGON_ITERATIONS", "3", 1, 6)),
		ArgonThreads:          uint8(r.Uint("ARGON_THREADS", "2", 1, 8)),
	}
	if len(c.CentrifugoTokenSecret) < 32 {
		r.Invalid("CENTRIFUGO_TOKEN_SECRET")
	}
	if c.JWTKid == "" {
		r.Invalid("JWT_KID")
	}
	return c, r.Err()
}
