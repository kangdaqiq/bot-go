package services

import (
	"log"
	"os"
	"sync"
	"time"

	"bot_wa/config"
)

// Session holds conversation state for a user
type Session struct {
	Type             string
	Action           string
	Step             string
	SearchResults    []Siswa
	TeacherID        uint
	TeacherName      string
	SelectedStudent  *Siswa
	SelectedStatus   string
	ExistingAtt      *Attendance
	BotMessageIDs    []BotMessage
	RegistrationType string // "siswa" or "ortu"
	Timestamp        time.Time
}

type BotMessage struct {
	ID     string
	ChatID string
}

var (
	sessions      = make(map[string]*Session)
	sessionsMu    sync.RWMutex
	sessionTimeout = 2 * time.Minute
)

func SetSession(phone string, session *Session) {
	session.Timestamp = time.Now()
	sessionsMu.Lock()
	sessions[phone] = session
	sessionsMu.Unlock()
}

func GetSession(phone string) *Session {
	sessionsMu.RLock()
	s, ok := sessions[phone]
	sessionsMu.RUnlock()
	if !ok {
		return nil
	}
	if time.Since(s.Timestamp) > sessionTimeout {
		sessionsMu.Lock()
		delete(sessions, phone)
		sessionsMu.Unlock()
		return nil
	}
	return s
}

func ClearSession(phone string) {
	sessionsMu.Lock()
	delete(sessions, phone)
	sessionsMu.Unlock()
}

func AddBotMessageID(phone, msgID, chatID string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s, ok := sessions[phone]
	if ok {
		s.BotMessageIDs = append(s.BotMessageIDs, BotMessage{ID: msgID, ChatID: chatID})
	}
}

func CleanupExpiredSessions() {
	sessionsMu.Lock()
	expired := []string{}
	for phone, s := range sessions {
		if time.Since(s.Timestamp) > sessionTimeout {
			expired = append(expired, phone)
		}
	}
	for _, phone := range expired {
		delete(sessions, phone)
	}
	sessionsMu.Unlock()

	for _, phone := range expired {
		msg := "⏳ *Sesi Berakhir*\n\nMaaf, sesi Anda telah berakhir karena tidak ada respons selama 2 menit.\n\nSilakan ulangi perintah Anda dari awal."
		SendMessage(phone, msg, "")
		log.Printf("⏰ Session expired for %s", phone)
	}
}

func StartSessionCleanup() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			CleanupExpiredSessions()
		}
	}()
}

// ─── WhatsApp Send ────────────────────────────────────────────────────────────

type SendResult struct {
	Success bool
	Data    map[string]interface{}
}

func SendMessage(phone, message, deviceID string) SendResult {
	if config.WaClient == nil {
		log.Println("⚠️ WaClient not initialized")
		return SendResult{Success: false}
	}

	// Normalize phone
	if !containsAt(phone) && len(phone) <= 15 {
		phone = NormalizePhone(phone)
	}

	if deviceID == "" {
		deviceID = "1"
	}

	endpoints := []string{"/send/message", "/api/send/message", "/api/send/text", "/send-message"}
	payload := map[string]string{"phone": phone, "message": message}

	for _, ep := range endpoints {
		var result map[string]interface{}
		resp, err := config.WaClient.R().
			SetBasicAuth(config.WaAPIUser, config.WaAPIPassword).
			SetHeader("Content-Type", "application/json").
			SetHeader("X-Device-Id", deviceID).
			SetBody(payload).
			SetResult(&result).
			Post(config.WaAPIURL + ep)

		if err != nil {
			continue
		}
		if resp.StatusCode() == 200 {
			log.Printf("✅ Message sent to %s via %s", phone, ep)
			return SendResult{Success: true, Data: result}
		}
	}
	log.Printf("❌ Failed to send message to %s", phone)
	return SendResult{Success: false}
}

func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

// ─── Resolve Device ID ────────────────────────────────────────────────────────

var (
	deviceCache     = make(map[string]string)
	deviceCacheMu   sync.RWMutex
	lastCacheTime   time.Time
)

func ResolveDeviceID(deviceID string) string {
	if isSimpleID(deviceID) {
		return deviceID
	}

	deviceCacheMu.RLock()
	cacheValid := time.Since(lastCacheTime) < 60*time.Second
	cached, inCache := deviceCache[deviceID]
	deviceCacheMu.RUnlock()

	if cacheValid && inCache {
		return cached
	}

	// Fetch device list from GoWA API (timeout 15 detik mengikuti Node.js)
	deviceFetchClient := config.WaClient.Clone()
	deviceFetchClient.SetTimeout(15 * time.Second)

	var apiResult struct {
		Results []struct {
			ID  string `json:"id"`
			JID string `json:"jid"`
		} `json:"results"`
	}
	_, err := deviceFetchClient.R().
		SetBasicAuth(config.WaAPIUser, config.WaAPIPassword).
		SetResult(&apiResult).
		Get(config.WaAPIURL + "/devices")

	deviceCacheMu.Lock()
	if err == nil && len(apiResult.Results) > 0 {
		for _, d := range apiResult.Results {
			if d.JID != "" {
				deviceCache[d.JID] = d.ID
				parts := splitAt(d.JID)
				if len(parts) == 2 {
					deviceCache[parts[0]] = d.ID
				}
			}
			if d.ID != "" {
				deviceCache[d.ID] = d.ID
			}
		}
		lastCacheTime = time.Now()
		log.Printf("✅ Device cache diperbarui dari GoWA API: %d device(s)", len(apiResult.Results))
	} else if err != nil {
		log.Printf("⚠️ GoWA API tidak bisa diakses: %v", err)
	}
	deviceCacheMu.Unlock()

	deviceCacheMu.RLock()
	id, ok := deviceCache[deviceID]
	deviceCacheMu.RUnlock()
	if ok {
		return id
	}

	// Fallback jika API gagal
	if envID := os.Getenv("WA_DEVICE_ID"); envID != "" {
		log.Printf("⚠️ Menggunakan WA_DEVICE_ID dari .env: %s", envID)
		return envID
	}
	if parts := splitAt(deviceID); len(parts) == 2 {
		log.Printf("⚠️ Fallback strip JID: %s -> %s", deviceID, parts[0])
		return parts[0]
	}
	return deviceID
}

func isSimpleID(s string) bool {
	if len(s) >= 5 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func splitAt(s string) []string {
	for i, c := range s {
		if c == '@' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
