package config

import (
	"log"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	WaAPIURL      string
	WaAPIUser     string
	WaAPIPassword string
	WaClient      *resty.Client
)

func InitWhatsApp() {
	WaAPIURL = os.Getenv("GOWA_API_BASE_URL")
	if WaAPIURL == "" {
		WaAPIURL = os.Getenv("WA_API_URL")
		if WaAPIURL == "" {
			WaAPIURL = "http://localhost:3000"
		}
	}

	WaAPIUser = os.Getenv("GOWA_API_USER")
	if WaAPIUser == "" {
		WaAPIUser = os.Getenv("WA_API_USER")
	}

	WaAPIPassword = os.Getenv("GOWA_API_PASS")
	if WaAPIPassword == "" {
		WaAPIPassword = os.Getenv("GOWA_API_PASSWORD")
		if WaAPIPassword == "" {
			WaAPIPassword = os.Getenv("WA_API_PASSWORD")
		}
	}

	WaClient = resty.New()
	WaClient.SetTimeout(10 * time.Second)
	
	log.Println("✅ WhatsApp API configuration loaded")
}
