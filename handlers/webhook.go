package handlers

import (
	"fmt"
	"log"
	"strings"

	"bot_wa/services"

	"github.com/gofiber/fiber/v2"
)

// â”€â”€â”€ Webhook Payload â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type MessagePayload struct {
	ChatID   string `json:"chat_id"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	Message  struct {
		Text string `json:"text"`
	} `json:"message"`
	Pushname string `json:"pushname"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

type WebhookBody struct {
	Event    string         `json:"event"`
	DeviceID string         `json:"device_id"`
	Payload  MessagePayload `json:"payload"`
	// flat format
	ChatID   string `json:"chat_id"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	Message  struct {
		Text string `json:"text"`
	} `json:"message"`
	Pushname string `json:"pushname"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

// â”€â”€â”€ Main Webhook Handler â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func HandleWebhook(c *fiber.Ctx) error {
	var wb WebhookBody
	if err := c.BodyParser(&wb); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON"})
	}

	// Determine raw device ID
	rawDeviceID := c.Get("X-Device-Id")
	if rawDeviceID == "" {
		rawDeviceID = c.Query("device_id")
	}
	if rawDeviceID == "" {
		rawDeviceID = "1"
	}

	// Resolve nested payload
	var payload MessagePayload
	if wb.Event == "message" && wb.Payload.From != "" {
		payload = wb.Payload
		if wb.DeviceID != "" {
			rawDeviceID = wb.DeviceID
		}
	} else {
		payload.ChatID = wb.ChatID
		payload.From = wb.From
		payload.FromName = wb.FromName
		payload.Message = wb.Message
		payload.Pushname = wb.Pushname
		payload.SenderID = wb.SenderID
		payload.Body = wb.Body
		if wb.DeviceID != "" {
			rawDeviceID = wb.DeviceID
		}
	}

	// Normalize flat body -> message.text
	if payload.Body != "" && payload.Message.Text == "" {
		payload.Message.Text = payload.Body
	}
	if payload.FromName != "" && payload.Pushname == "" {
		payload.Pushname = payload.FromName
	}
	if payload.SenderID == "" {
		payload.SenderID = payload.From
	}

	// Resolve device ID
	deviceID := services.ResolveDeviceID(rawDeviceID)
	log.Printf("ðŸ”Œ Resolved Device ID: %s -> %s", rawDeviceID, deviceID)

	// Check if message received by superadmin device ID
	if rawDeviceID == "superadmin" || deviceID == "superadmin" {
		log.Printf("⚠️ Ignoring message received by superadmin device")
		return c.JSON(fiber.Map{"success": true, "message": "Superadmin messages ignored"})
	}

	// Check if text message
	body := payload.Message.Text
	if body == "" {
		log.Println("âš ï¸ Not a text message or missing message field")
		return c.JSON(fiber.Map{"success": true, "message": "Non-text message ignored"})
	}

	// Check bot enabled
	if !services.IsBotEnabled(deviceID) {
		log.Printf("ðŸ¤– Bot dinonaktifkan untuk device: %s", deviceID)
		return c.JSON(fiber.Map{"success": true, "message": "Bot disabled"})
	}

	// Extract phone number
	phoneNumber := extractPhone(payload.From)
	log.Printf("ðŸ“ž Extracted phone number: %s", phoneNumber)

	// Group detection
	chatID := payload.ChatID
	isGroup := strings.Contains(chatID, "@g.us") ||
		strings.Contains(chatID, "@lid") ||
		chatID != payload.SenderID

	// Abaikan semua pesan dari group â€” bot hanya merespon pesan private
	if isGroup {
		log.Printf("â­ï¸ Ignoring group message from chat: %s", chatID)
		return c.JSON(fiber.Map{"success": true, "message": "Group messages ignored"})
	}

	log.Printf("âœ… Private message detected from %s", phoneNumber)

	// Record last seen interaction
	go services.UpdateLastSeen(phoneNumber)

	// Check if teacher
	teacher, err := services.GetTeacherByPhone(phoneNumber, deviceID)
	if err != nil {
		log.Printf("❌ Error getting teacher: %v", err)
	}

	if teacher != nil {
		log.Printf("👨‍🏫 Teacher found: %s", teacher.Nama)
		if !services.HasTeacherBotAccess(teacher.ID) {
			log.Printf("🚫 Guru %s tidak memiliki akses bot", teacher.Nama)
			return c.JSON(fiber.Map{"success": true, "message": "Teacher bot access denied"})
		}

		result := handleTeacherMessage(phoneNumber, body, teacher, deviceID)
		return c.JSON(result)
	}

	// Non-teacher private â€” cek apakah sedang dalam sesi registrasi
	log.Printf("ðŸ“‹ Non-teacher message from %s, checking registration session", phoneNumber)
	schoolName := services.GetSchoolName(deviceID)
	result := handlePublicMessage(phoneNumber, body, deviceID, schoolName)
	return c.JSON(result)
}

// â”€â”€â”€ Teacher Message Handler â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func handleTeacherMessage(replyTo, body string, teacher *services.Guru, deviceID string) fiber.Map {
	phoneNumber := teacher.NoWa
	session := services.GetSession(phoneNumber)
	cmd := services.ParseTeacherCommand(body)
	log.Printf("âš¡ Teacher command: %s, Search: %s, Option: %d", cmd.Command, cmd.SearchTerm, cmd.Option)

	var responseMessage string

	switch {
	case cmd.Command == "teacher_help":
		services.ClearSession(phoneNumber)
		schoolName := services.GetSchoolName(deviceID)
		responseMessage = services.GenerateTeacherHelpMessage(teacher.Nama, schoolName)

	case cmd.Command == "search_student":
		students, _ := services.SearchStudentsByName(cmd.SearchTerm, deviceID)
		responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
		if len(students) > 0 {
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "create", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "edit_attendance":
		students, _ := services.SearchStudentsWithAttendanceToday(cmd.SearchTerm, deviceID)
		if len(students) == 0 {
			responseMessage = services.GenerateNoAttendanceFoundMessage(cmd.SearchTerm)
		} else {
			responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "edit", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "delete_attendance":
		students, _ := services.SearchStudentsWithAttendanceToday(cmd.SearchTerm, deviceID)
		if len(students) == 0 {
			responseMessage = services.GenerateNoAttendanceFoundMessage(cmd.SearchTerm)
		} else {
			responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "delete", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "quick_checkin":
		students, _ := services.SearchStudentsByName(cmd.SearchTerm, deviceID)
		responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
		if len(students) > 0 {
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "quick_checkin", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "quick_checkout":
		students, _ := services.SearchStudentsWithAttendanceToday(cmd.SearchTerm, deviceID)
		if len(students) == 0 {
			responseMessage = services.GenerateNoAttendanceForCheckout(cmd.SearchTerm)
		} else {
			responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "quick_checkout", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "recap_student":
		students, _ := services.SearchStudentsByName(cmd.SearchTerm, deviceID)
		responseMessage = services.GenerateStudentSearchResults(students, cmd.SearchTerm)
		if len(students) > 0 {
			services.SetSession(phoneNumber, &services.Session{
				Type: "teacher", Action: "recap_student", Step: "select_student",
				SearchResults: students, TeacherID: teacher.ID, TeacherName: teacher.Nama,
			})
		}

	case cmd.Command == "search_contact":
		results, _ := services.SearchStudentContact(cmd.SearchTerm, deviceID)
		responseMessage = services.GenerateContactInfo(results, cmd.SearchTerm)

	case cmd.Command == "select_option" && session != nil:
		responseMessage = handleSelectOption(cmd.Option, session, phoneNumber, deviceID, teacher)

	case cmd.Command == "confirm_yes" && session != nil && session.Step == "confirm_delete":
		services.DeleteAttendanceToday(session.SelectedStudent.ID)
		responseMessage = services.GenerateDeleteSuccessMessage(session.SelectedStudent.Nama)
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_no" && session != nil && session.Step == "confirm_delete":
		responseMessage = "âŒ *Penghapusan Dibatalkan*\n\nAbsensi tidak jadi dihapus.\n\nKetik `help` untuk melihat perintah lainnya."
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_yes" && session != nil && session.Step == "confirm_replace_create":
		responseMessage = services.GenerateStatusSelectionMessage(session.SelectedStudent.Nama)
		session.Step = "select_status"
		services.SetSession(phoneNumber, session)

	case cmd.Command == "confirm_no" && session != nil && session.Step == "confirm_replace_create":
		responseMessage = "âŒ *Perubahan Dibatalkan*\n\nAbsensi tidak jadi diubah.\n\nKetik `help` untuk melihat perintah lainnya."
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_yes" && session != nil && session.Step == "confirm_replace_checkin":
		services.QuickCheckin(session.SelectedStudent.ID, session.TeacherID, session.TeacherName, deviceID)
		responseMessage = services.GenerateQuickCheckinSuccess(session.SelectedStudent.Nama, session.SelectedStudent.NamaKelas)
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_no" && session != nil && session.Step == "confirm_replace_checkin":
		responseMessage = "âŒ *Perubahan Dibatalkan*\n\nJam masuk tidak jadi diubah.\n\nKetik `help` untuk melihat perintah lainnya."
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_yes" && session != nil && session.Step == "confirm_replace_checkout":
		result := services.QuickCheckout(session.SelectedStudent.ID, session.TeacherName, deviceID)
		if result.Success {
			responseMessage = services.GenerateQuickCheckoutSuccess(session.SelectedStudent.Nama, session.SelectedStudent.NamaKelas, result.JamMasuk)
		} else {
			responseMessage = services.GenerateNoAttendanceForCheckout(session.SelectedStudent.Nama)
		}
		services.ClearSession(phoneNumber)

	case cmd.Command == "confirm_no" && session != nil && session.Step == "confirm_replace_checkout":
		responseMessage = "âŒ *Perubahan Dibatalkan*\n\nJam pulang tidak jadi diubah.\n\nKetik `help` untuk melihat perintah lainnya."
		services.ClearSession(phoneNumber)

	case session != nil && session.Step == "input_keterangan" && session.Action == "create":
		keterangan := strings.TrimSpace(body)
		if keterangan == "" {
			responseMessage = "âŒ *Keterangan tidak boleh kosong*\n\nSilakan ketik keterangan untuk absensi ini."
		} else {
			services.CreateManualAttendance(session.SelectedStudent.ID, session.SelectedStatus, session.TeacherName, keterangan, deviceID, session.TeacherID)
			responseMessage = services.GenerateAttendanceConfirmation(session.SelectedStudent.Nama, session.SelectedStudent.NamaKelas, session.SelectedStatus, keterangan)
			services.ClearSession(phoneNumber)
		}

	case session != nil && session.Step == "edit_keterangan" && session.Action == "edit":
		keterangan := strings.TrimSpace(body)
		if keterangan == "" {
			responseMessage = "âŒ *Keterangan tidak boleh kosong*\n\nSilakan ketik keterangan baru."
		} else {
			services.UpdateAttendanceKeterangan(session.SelectedStudent.ID, keterangan, session.TeacherName, deviceID)
			responseMessage = services.GenerateEditSuccessMessage(session.SelectedStudent.Nama, "keterangan", keterangan)
			services.ClearSession(phoneNumber)
		}

	default:
		if session != nil {
			// Ada sesi aktif tapi input tidak dikenali â†’ kirim alert sesuai step,
			// JANGAN hapus sesi agar guru bisa melanjutkan dari langkah yang sama.
			responseMessage = services.GenerateSessionAlertMessage(session)
		} else {
			// Tidak ada sesi â†’ tampilkan menu bantuan seperti biasa
			schoolName := services.GetSchoolName(deviceID)
			responseMessage = services.GenerateTeacherHelpMessage(teacher.Nama, schoolName)
		}
	}

	sentResult := services.SendMessage(replyTo, responseMessage, deviceID)
	log.Printf("ðŸ“¤ Sending response to teacher at %s", replyTo)

	if currentSession := services.GetSession(phoneNumber); currentSession != nil && sentResult.Success {
		if data, ok := sentResult.Data["results"]; ok {
			if resultsMap, ok := data.(map[string]interface{}); ok {
				if msgID, ok := resultsMap["message_id"].(string); ok && msgID != "" {
					services.AddBotMessageID(phoneNumber, msgID, replyTo)
				}
			}
		}
	}

	return fiber.Map{"success": true, "message": "Response sent", "user": teacher.Nama, "command": cmd.Command}
}

// â”€â”€â”€ Select Option Handler â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func handleSelectOption(option int, session *services.Session, phoneNumber, deviceID string, teacher *services.Guru) string {
	switch session.Step {
	case "select_student":
		idx := option - 1
		if idx < 0 || idx >= len(session.SearchResults) {
			return services.GenerateInvalidSelectionMessage(len(session.SearchResults))
		}
		selected := session.SearchResults[idx]
		session.SelectedStudent = &selected

		switch session.Action {
		case "create":
			existing, _ := services.GetTodayAttendance(selected.ID)
			if existing != nil {
				services.SetSession(phoneNumber, &services.Session{
					Type: session.Type, Action: session.Action, Step: "confirm_replace_create",
					TeacherID: session.TeacherID, TeacherName: session.TeacherName,
					SelectedStudent: &selected, ExistingAtt: existing,
				})
				return services.GenerateAttendanceExistsConfirmation(&selected, existing)
			}
			session.Step = "select_status"
			services.SetSession(phoneNumber, session)
			return services.GenerateStatusSelectionMessage(selected.Nama)

		case "edit":
			session.Step = "select_edit_type"
			services.SetSession(phoneNumber, session)
			return services.GenerateEditOptionsMessage(selected.Nama)

		case "delete":
			att, _ := services.GetStudentAttendanceToday(selected.ID)
			if att == nil {
				services.ClearSession(phoneNumber)
				return services.GenerateNoAttendanceFoundMessage(selected.Nama)
			}
			session.Step = "confirm_delete"
			session.ExistingAtt = att
			services.SetSession(phoneNumber, session)
			return services.GenerateDeleteConfirmationMessage(&selected, att)

		case "quick_checkin":
			existing, _ := services.GetTodayAttendance(selected.ID)
			if existing != nil {
				session.Step = "confirm_replace_checkin"
				session.ExistingAtt = existing
				services.SetSession(phoneNumber, session)
				return services.GenerateCheckinExistsConfirmation(&selected, existing)
			}
			services.QuickCheckin(selected.ID, session.TeacherID, session.TeacherName, deviceID)
			services.ClearSession(phoneNumber)
			return services.GenerateQuickCheckinSuccess(selected.Nama, selected.NamaKelas)

		case "quick_checkout":
			existing, _ := services.GetTodayAttendance(selected.ID)
			if existing == nil {
				services.ClearSession(phoneNumber)
				return services.GenerateNoAttendanceForCheckout(selected.Nama)
			}
			if existing.JamPulang != "" {
				session.Step = "confirm_replace_checkout"
				session.ExistingAtt = existing
				services.SetSession(phoneNumber, session)
				return services.GenerateCheckoutExistsConfirmation(&selected, existing)
			}
			result := services.QuickCheckout(selected.ID, session.TeacherName, deviceID)
			services.ClearSession(phoneNumber)
			if result.Success {
				return services.GenerateQuickCheckoutSuccess(selected.Nama, selected.NamaKelas, result.JamMasuk)
			}
			return services.GenerateNoAttendanceForCheckout(selected.Nama)

		case "recap_student":
			stats, _ := services.GetAttendanceRecap(selected.ID, "month")
			services.ClearSession(phoneNumber)
			return services.GenerateRecapMessage(&selected, stats, "month")
		}

	case "select_status":
		statusMap := map[int]string{1: "H", 2: "I", 3: "S", 4: "A"}
		status, ok := statusMap[option]
		if !ok {
			return services.GenerateInvalidSelectionMessage(4)
		}
		if status == "H" {
			ket := "Hadir (dicatat oleh " + session.TeacherName + ")"
			services.CreateManualAttendance(session.SelectedStudent.ID, status, session.TeacherName, ket, deviceID, session.TeacherID)
			services.ClearSession(phoneNumber)
			return services.GenerateAttendanceConfirmation(session.SelectedStudent.Nama, session.SelectedStudent.NamaKelas, status, ket)
		}
		session.Step = "input_keterangan"
		session.SelectedStatus = status
		services.SetSession(phoneNumber, session)
		return services.GenerateKeteranganInputMessage(session.SelectedStudent.Nama, status)

	case "select_edit_type":
		switch option {
		case 1:
			session.Step = "edit_status"
			services.SetSession(phoneNumber, session)
			return services.GenerateStatusSelectionMessage(session.SelectedStudent.Nama)
		case 2:
			session.Step = "edit_keterangan"
			services.SetSession(phoneNumber, session)
			return "ðŸ“ *Edit Keterangan*\n\nSiswa: *" + session.SelectedStudent.Nama + "*\n\nSilakan ketik keterangan baru.\n\nðŸ’¡ _Ketik keterangan baru_"
		default:
			return services.GenerateInvalidSelectionMessage(2)
		}

	case "edit_status":
		statusMap := map[int]string{1: "H", 2: "I", 3: "S", 4: "A"}
		status, ok := statusMap[option]
		if !ok {
			return services.GenerateInvalidSelectionMessage(4)
		}
		services.UpdateAttendanceStatus(session.SelectedStudent.ID, status, session.TeacherName, deviceID)
		services.ClearSession(phoneNumber)
		statusLabels := map[string]string{"H": "Hadir", "I": "Izin", "S": "Sakit", "A": "Alpha"}
		return services.GenerateEditSuccessMessage(session.SelectedStudent.Nama, "status", statusLabels[status])
	}

	return services.GenerateInvalidSelectionMessage(len(session.SearchResults))
}

// â”€â”€â”€ Extract phone from 'from' field â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func extractPhone(from string) string {
	if strings.Contains(from, ":") {
		return strings.Split(from, ":")[0]
	}
	if strings.Contains(from, "@") {
		return strings.Split(from, "@")[0]
	}
	return from
}

// --- Public (Siswa/Ortu) Message Handler ---

func handlePublicMessage(phoneNumber, body, deviceID, schoolName string) fiber.Map {
	session := services.GetSession(phoneNumber)
	cmd := services.ParsePublicCommand(body)
	log.Printf("Public command: %s from %s", cmd.Command, phoneNumber)

	var responseMessage string

	switch {
	// Mulai pendaftaran siswa
	case cmd.Command == "register_student":
		services.SetSession(phoneNumber, &services.Session{
			Type:             "public",
			Action:           "register",
			Step:             "register_ask_nis",
			RegistrationType: "siswa",
		})
		responseMessage = services.GenerateRegisterAskNISMessage("siswa")

	// Mulai pendaftaran ortu
	case cmd.Command == "register_parent":
		services.SetSession(phoneNumber, &services.Session{
			Type:             "public",
			Action:           "register",
			Step:             "register_ask_nis",
			RegistrationType: "ortu",
		})
		responseMessage = services.GenerateRegisterAskNISMessage("ortu")

	// Dalam sesi registrasi -- delegasikan ke state machine
	case session != nil && session.Action == "register":
		responseMessage = handleRegistrationMessage(phoneNumber, body, session, deviceID, schoolName)

	// Tidak ada sesi -- tampilkan selamat datang
	default:
		// Mengabaikan pesan dari nomor tidak dikenal agar bot tidak terdeteksi sebagai spam
		log.Printf("Mengabaikan pesan dari %s agar tidak terdeteksi spam", phoneNumber)
		return fiber.Map{"success": true, "message": "Ignored to prevent spam", "command": cmd.Command}
	}

	services.SendMessage(phoneNumber, responseMessage, deviceID)
	log.Printf("Registration response sent to %s", phoneNumber)
	return fiber.Map{"success": true, "message": "Registration response sent", "command": cmd.Command}
}

func handleRegistrationMessage(phoneNumber, body string, session *services.Session, deviceID, schoolName string) string {
	switch session.Step {

	// Langkah 1: User kirim NIS
	case "register_ask_nis":
		nis := strings.TrimSpace(body)
		if nis == "" {
			return services.GenerateRegisterSessionAlert("register_ask_nis")
		}
		student, err := services.GetStudentByNIS(nis, deviceID)
		if err != nil {
			log.Printf("DB error on NIS lookup: %v", err)
			return services.GenerateErrorMessage()
		}
		if student == nil {
			return services.GenerateRegisterNISNotFoundMessage(nis, session.RegistrationType)
		}
		session.SelectedStudent = student
		session.Step = "register_ask_tgl_lahir"
		services.SetSession(phoneNumber, session)
		return services.GenerateRegisterAskTglLahirMessage(student, session.RegistrationType)

	// Langkah 2: User kirim tanggal lahir
	case "register_ask_tgl_lahir":
		input := strings.TrimSpace(body)
		if !isValidDateInput(input) {
			return services.GenerateRegisterSessionAlert("register_ask_tgl_lahir")
		}
		normalized := normalizeDateInput(input)
		if session.SelectedStudent == nil {
			services.ClearSession(phoneNumber)
			return services.GenerateRegisterSessionAlert("")
		}
		dbDate := strings.TrimSpace(session.SelectedStudent.TglLahir)
		if len(dbDate) > 10 {
			dbDate = dbDate[:10]
		}
		if normalized != dbDate {
			log.Printf("TglLahir mismatch for NIS %s: input=%s db=%s", session.SelectedStudent.NIS, normalized, dbDate)
			services.ClearSession(phoneNumber)
			return services.GenerateRegisterTglLahirWrongMessage(session.RegistrationType)
		}
		isOrtu := session.RegistrationType == "ortu"
		err := services.UpdateStudentPhone(session.SelectedStudent.ID, services.NormalizePhone(phoneNumber), isOrtu)
		if err != nil {
			log.Printf("Error saving phone: %v", err)
			services.ClearSession(phoneNumber)
			return services.GenerateErrorMessage()
		}
		studentName := session.SelectedStudent.Nama
		regType := session.RegistrationType
		services.ClearSession(phoneNumber)
		return services.GenerateRegisterSuccessMessage(studentName, regType, schoolName)
	}

	return services.GenerateRegisterSessionAlert(session.Step)
}

// isValidDateInput menerima format DD-MM-YYYY atau DD/MM/YYYY
func isValidDateInput(s string) bool {
	s = strings.ReplaceAll(s, "/", "-")
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return len(parts[2]) == 4
}

// normalizeDateInput konversi DD-MM-YYYY atau DD/MM/YYYY ke YYYY-MM-DD
func normalizeDateInput(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return s
	}
	dd := fmt.Sprintf("%02s", parts[0])
	mm := fmt.Sprintf("%02s", parts[1])
	yyyy := parts[2]
	return yyyy + "-" + mm + "-" + dd
}
