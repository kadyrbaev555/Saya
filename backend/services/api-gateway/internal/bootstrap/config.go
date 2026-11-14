package bootstrap

import "saya/config"

type Config struct {
	AuthAddress       string
	SurveyAddress     string
	AssessmentAddress string
	AIAddress         string
	JWTPublic         string
	JWTKid            string
	JWTPublicKeys     string
	WebOrigin         string
	CookieSecure      bool
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		AuthAddress:       r.Address("AUTH_ADDRESS", "auth:9000"),
		SurveyAddress:     r.Address("SURVEY_ADDRESS", "survey:9000"),
		AssessmentAddress: r.Address("ASSESSMENT_ADDRESS", "assessment:9000"),
		AIAddress:         r.Address("AI_ADDRESS", "ai:9000"),
		JWTPublic:         r.Key("JWT_PUBLIC"),
		JWTKid:            r.String("JWT_KID", "local-1"),
		JWTPublicKeys:     r.String("JWT_PUBLIC_KEYS", ""),
		WebOrigin:         r.URL("WEB_ORIGIN", "http", "https"),
		CookieSecure:      r.Bool("COOKIE_SECURE", "true"),
	}

	return c, r.Err()
}
