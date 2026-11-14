// dev-env creates missing service files from existing local infrastructure secrets.
package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func run() error {
	values, e := godotenv.Read("deployments/local/.env")
	if e != nil {
		return fmt.Errorf("read infrastructure .env first (make dev-init)")
	}
	templates := map[string]map[string]string{
		"auth-service": {
			"DATABASE_URL":            "postgres://saya_auth:${DB_PASSWORD}@postgres:5432/saya_auth?sslmode=disable",
			"CENTRIFUGO_TOKEN_SECRET": "${CENTRIFUGO_TOKEN_SECRET:?run make dev-init}",
			"JWT_KID":                 "${JWT_KID:-local-1}",
			"ARGON_MEMORY_KIB":        "${ARGON_MEMORY_KIB:-65536}",
			"ARGON_ITERATIONS":        "${ARGON_ITERATIONS:-3}",
			"ARGON_THREADS":           "${ARGON_THREADS:-2}",
			"JWT_SEED":                "${JWT_SEED:?run make dev-init}",
		},
		"api-gateway": {
			"AUTH_ADDRESS":       "auth:9000",
			"SURVEY_ADDRESS":     "survey:9000",
			"ASSESSMENT_ADDRESS": "assessment:9000",
			"JWT_KID":            "${JWT_KID:-local-1}",
			"JWT_PUBLIC_KEYS":    "${JWT_PUBLIC_KEYS:-}",
			"JWT_PUBLIC":         "${JWT_PUBLIC:?run make dev-init}",
			"WEB_ORIGIN":         "http://localhost:3000",
			"COOKIE_SECURE":      "false",
			"AI_ADDRESS":         "ai:9000",
		},
		"survey-service": {
			"DATABASE_URL": "postgres://saya_survey:${SURVEY_DB_PASSWORD}@postgres:5432/saya_survey?sslmode=disable",
			"RABBIT_URL":   "amqp://saya:${RABBIT_PASSWORD}@rabbitmq:5672/",
		},
		"assessment-service": {
			"DATABASE_URL":       "postgres://saya_assessment:${ASSESSMENT_DB_PASSWORD}@postgres:5432/saya_assessment?sslmode=disable",
			"RABBIT_URL":         "amqp://saya:${RABBIT_PASSWORD}@rabbitmq:5672/",
			"SURVEY_ADDRESS":     "survey:9000",
			"CENTRIFUGO_URL":     "http://centrifugo:8000",
			"CENTRIFUGO_API_KEY": "${CENTRIFUGO_API_KEY}",
		},
		"ai-service": {
			"ASSESSMENT_ADDRESS": "assessment:9000",
			"RABBIT_URL":         "amqp://saya:${RABBIT_PASSWORD}@rabbitmq:5672/",
			"GEMINI_API_KEY":     "${GEMINI_API_KEY:-}",
			"GEMINI_MODEL":       "${GEMINI_MODEL:-}",
			"LLM_PROVIDER":       "${LLM_PROVIDER:-mock}",
			"LLM_ENDPOINT":       "${LLM_ENDPOINT:-}",
			"LLM_API_KEY":        "${LLM_API_KEY:-}",
		},
	}
	pattern := regexp.MustCompile(`\$\{([A-Z_]+)(?::([-?])([^}]*))?\}`)
	for service, entries := range templates {
		path := filepath.Join("services", service, ".env")
		if _, e := os.Stat(path); e == nil {
			continue
		} else if !os.IsNotExist(e) {
			return fmt.Errorf("cannot inspect service env")
		}
		output := map[string]string{}
		for key, template := range entries {
			missing := false
			value := pattern.ReplaceAllStringFunc(template, func(s string) string {
				parts := pattern.FindStringSubmatch(s)
				v := values[parts[1]]
				if v != "" {
					return v
				}
				if parts[2] == "-" {
					return parts[3]
				}
				missing = true
				return ""
			})
			if missing {
				return fmt.Errorf("missing infrastructure setting for %s: %s", service, key)
			}
			output[key] = value
		}
		keys := []string{}
		for k := range output {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var text strings.Builder
		text.WriteString("# Local service configuration. Never commit this file.\n")
		for _, k := range keys {
			v := output[k]
			if strings.ContainsAny(v, "'\r\n") {
				return fmt.Errorf("unsupported newline or quote in %s", k)
			}
			fmt.Fprintf(&text, "%s='%s'\n", k, v)
		}
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return fmt.Errorf("cannot create %s env", service)
		}
		_, e = f.WriteString(text.String())
		ce := f.Close()
		if e != nil || ce != nil {
			return fmt.Errorf("cannot write %s env", service)
		}
		fmt.Println("Created", path)
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
