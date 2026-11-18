package bootstrap

import "saya/config"

type Config struct {
	TestMerchant, TestPaymentKey, TestPayoutKey, PaymentCallback string
	FCMCredentials                                               string
	DatabaseURL                                                  string
	RabbitURL                                                    string
	SurveyAddress                                                string
	CentrifugoURL                                                string
	CentrifugoAPIKey                                             string
}

func LoadConfig() (Config, error) {
	r, e := config.Open()
	if e != nil {
		return Config{}, e
	}
	c := Config{
		TestMerchant: r.String("FREEDOMPAY_TEST_MERCHANT_ID", ""), TestPaymentKey: r.String("FREEDOMPAY_TEST_SECRET", ""), TestPayoutKey: r.String("FREEDOMPAY_TEST_PAYOUT_SECRET", ""), PaymentCallback: r.String("FREEDOMPAY_RESULT_URL", ""),
		FCMCredentials:   r.String("FCM_SERVICE_ACCOUNT_B64", ""),
		DatabaseURL:      r.URL("DATABASE_URL", "postgres", "postgresql"),
		RabbitURL:        r.URL("RABBIT_URL", "amqp", "amqps"),
		SurveyAddress:    r.Address("SURVEY_ADDRESS", "survey:9000"),
		CentrifugoURL:    r.URL("CENTRIFUGO_URL", "http", "https"),
		CentrifugoAPIKey: r.Required("CENTRIFUGO_API_KEY"),
	}

	return c, r.Err()
}
