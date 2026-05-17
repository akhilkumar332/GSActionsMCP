package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultCSRFKey = "01234567890123456789012345678901"

type runtimeConfig struct {
	Env                 string
	LocalDev            bool
	DatabaseURL         string
	RedisURL            string
	BaseURL             string
	CSRFKey             string
	EncryptionKey       string
	StripeAPIKey        string
	StripeWebhookSecret string
	StripeProPriceID    string
	StoreLLMResponses   bool
	MaxLLMResponseChars int
}

func loadRuntimeConfigFromEnv() (runtimeConfig, error) {
	cfg := runtimeConfig{
		Env:                 os.Getenv("ENV"),
		LocalDev:            os.Getenv("LOCAL_DEV") == "true",
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		BaseURL:             os.Getenv("BASE_URL"),
		CSRFKey:             os.Getenv("CSRF_KEY"),
		EncryptionKey:       os.Getenv("ENCRYPTION_KEY"),
		StripeAPIKey:        os.Getenv("STRIPE_API_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeProPriceID:    os.Getenv("STRIPE_PRO_PRICE_ID"),
		StoreLLMResponses:   envBool("STORE_LLM_RESPONSES", true),
		MaxLLMResponseChars: envInt("MAX_LLM_RESPONSE_CHARS", 4000),
	}

	if cfg.MaxLLMResponseChars < 256 {
		cfg.MaxLLMResponseChars = 256
	}

	if !cfg.productionMode() {
		return cfg, nil
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required in production")
	}
	if cfg.RedisURL == "" {
		return cfg, fmt.Errorf("REDIS_URL is required in production")
	}
	if cfg.BaseURL == "" {
		return cfg, fmt.Errorf("BASE_URL is required in production")
	}
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return cfg, fmt.Errorf("BASE_URL must be an absolute URL in production")
	}
	if baseURL.Scheme != "https" {
		return cfg, fmt.Errorf("BASE_URL must use https in production")
	}

	if len(cfg.CSRFKey) < 32 || cfg.CSRFKey == defaultCSRFKey {
		return cfg, fmt.Errorf("CSRF_KEY must be at least 32 bytes and must not use the insecure default in production")
	}

	if len(cfg.EncryptionKey) != 64 {
		return cfg, fmt.Errorf("ENCRYPTION_KEY must be a 64-character hex string in production")
	}
	keyBytes, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil || len(keyBytes) != 32 {
		return cfg, fmt.Errorf("ENCRYPTION_KEY must decode to 32 bytes in production")
	}

	hasAnyStripe := cfg.StripeAPIKey != "" || cfg.StripeWebhookSecret != "" || cfg.StripeProPriceID != ""
	if hasAnyStripe {
		if cfg.StripeAPIKey == "" || cfg.StripeWebhookSecret == "" || cfg.StripeProPriceID == "" {
			return cfg, fmt.Errorf("STRIPE_API_KEY, STRIPE_WEBHOOK_SECRET, and STRIPE_PRO_PRICE_ID must all be set together in production")
		}
	}

	return cfg, nil
}

func (c runtimeConfig) productionMode() bool {
	return c.Env == "production" && !c.LocalDev
}

func (c runtimeConfig) secureCookies() bool {
	return c.Env == "production" && !c.LocalDev
}

func (c runtimeConfig) csrfTrustedOrigins() []string {
	// gorilla/csrf TrustedOrigins should be scheme://host[:port]

	origins := []string{
		"http://localhost:8080",
		"https://localhost:8080",
		"http://127.0.0.1:8080",
		"https://127.0.0.1:8080",
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Common dev ports (Vite, etc.)
	devPorts := []string{port, "5173", "3000", "3001"}

	// Automatically trust all local network interfaces in dev
	if c.LocalDev {
		addrs, err := net.InterfaceAddrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ip := ipnet.IP.String()
						origins = append(origins, "http://"+ip)
						origins = append(origins, "https://"+ip)
						for _, p := range devPorts {
							origins = append(origins, "http://"+ip+":"+p)
							origins = append(origins, "https://"+ip+":"+p)
						}
					}
				}
			}
		}

		// Always trust localhost/127.0.0.1 with dev ports
		for _, p := range devPorts {
			origins = append(origins, "http://localhost:"+p)
			origins = append(origins, "https://localhost:"+p)
			origins = append(origins, "http://127.0.0.1:"+p)
			origins = append(origins, "https://127.0.0.1:"+p)
			origins = append(origins, "http://[::1]:"+p)
			origins = append(origins, "https://[::1]:"+p)
		}

		// Hardcode common local network ranges
		commonIPs := []string{"192.168.0.26", "192.168.1.1", "10.0.0.1", "172.17.0.1", "172.18.0.1"}
		for _, ip := range commonIPs {
			origins = append(origins, "http://"+ip)
			origins = append(origins, "https://"+ip)
			for _, p := range devPorts {
				origins = append(origins, "http://"+ip+":"+p)
				origins = append(origins, "https://"+ip+":"+p)
			}
		}
	}

	// Add from environment variable
	if extra := os.Getenv("CSRF_TRUSTED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			if !strings.Contains(o, "://") {
				origins = append(origins, "http://"+o)
				origins = append(origins, "https://"+o)
				if !strings.Contains(o, ":") {
					origins = append(origins, "http://"+o+":"+port)
					origins = append(origins, "https://"+o+":"+port)
				}
			} else {
				origins = append(origins, o)
			}
		}
	}

	if c.BaseURL != "" {
		parsed, err := url.Parse(c.BaseURL)
		if err == nil && parsed.Host != "" {
			cleanBase := strings.TrimSuffix(c.BaseURL, "/")
			origins = append(origins, cleanBase)

			hostWithPort := parsed.Host
			hostOnly := parsed.Hostname()

			origins = append(origins, "http://"+hostWithPort)
			origins = append(origins, "https://"+hostWithPort)
			origins = append(origins, "http://"+hostOnly)
			origins = append(origins, "https://"+hostOnly)

			if c.LocalDev {
				origins = append(origins, "http://"+hostOnly+":"+port)
				origins = append(origins, "https://"+hostOnly+":"+port)
			}
		}
	}

	// Deduplicate and clean
	unique := make(map[string]bool)
	var result []string
	for _, o := range origins {
		o = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(o, "/")))
		if o != "" && !unique[o] {
			unique[o] = true
			result = append(result, o)
		}
	}

	return result
}

func envBool(name string, fallback bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
