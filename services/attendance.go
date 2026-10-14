package services

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"bot_wa/config"

	"gorm.io/gorm"
)

// ─── Models ───────────────────────────────────────────────────────────────────

type Siswa struct {
	ID            uint       `gorm:"column:id"`
	Nama          string     `gorm:"column:nama"`
	NIS           string     `gorm:"column:nis"`
	NoWa          string     `gorm:"column:no_wa"`
	WaOrtu        string     `gorm:"column:wa_ortu"`
	TglLahir      string     `gorm:"column:tgl_lahir"`
	KelasID       uint       `gorm:"column:kelas_id"`
	SchoolID      uint       `gorm:"column:school_id"`
	NamaKelas     string     `gorm:"column:nama_kelas"`
	LastSeenSiswa *time.Time `gorm:"column:last_seen_siswa"`
	LastSeenOrtu  *time.Time `gorm:"column:last_seen_ortu"`
}

type Guru struct {
	ID        uint       `gorm:"column:id"`
	Nama      string     `gorm:"column:nama"`
	NoWa      string     `gorm:"column:no_wa"`
	SchoolID  uint       `gorm:"column:school_id"`
	BotAccess bool       `gorm:"column:bot_access"`
	LastSeen  *time.Time `gorm:"column:last_seen"`
}

type Attendance struct {
	ID                   uint   `gorm:"column:id"`
	StudentID            uint   `gorm:"column:student_id"`
	Tanggal              string `gorm:"column:tanggal"`
	JamMasuk             string `gorm:"column:jam_masuk"`
	JamPulang            string `gorm:"column:jam_pulang"`
	Status               string `gorm:"column:status"`
	Keterangan           string `gorm:"column:keterangan"`
	CheckedInByTeacherID *uint  `gorm:"column:checked_in_by_teacher_id"`
	TotalSeconds         int    `gorm:"column:total_seconds"`
	Nama                 string `gorm:"column:nama"`
	NamaKelas            string `gorm:"column:nama_kelas"`
}

type AttendanceStats struct {
	Total   int
	Hadir   int
	Izin    int
	Sakit   int
	Alpha   int
	Bolos   int
	Records []Attendance
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func NormalizePhone(phone string) string {
	// Strip non-digits
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	if strings.HasPrefix(cleaned, "0") {
		return "62" + cleaned[1:]
	}
	if !strings.HasPrefix(cleaned, "62") {
		return "62" + cleaned
	}
	return cleaned
}

func phoneVariants(phone string) []string {
	norm := NormalizePhone(phone)
	var variants []string
	variants = append(variants, norm)
	if strings.HasPrefix(norm, "62") {
		variants = append(variants, "0"+norm[2:])
		variants = append(variants, norm[2:])
	}
	return variants
}

func today() string {
	loc := GetJakartaLocation()
	return time.Now().In(loc).Format("2006-01-02")
}

func nowTime() string {
	loc := GetJakartaLocation()
	return time.Now().In(loc).Format("15:04:05")
}

// ─── Attendance Service Functions ─────────────────────────────────────────────

func GetTeacherByPhone(phone string, schoolID string) (*Guru, error) {
	variants := phoneVariants(phone)
	var guru Guru
	result := config.DB.Table("guru").
		Where("no_wa IN ? AND school_id = ?", variants, schoolID).
		First(&guru)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &guru, nil
}

func GetStudentByPhone(phone string, schoolID string) (*Siswa, error) {
	variants := phoneVariants(phone)
	var siswa Siswa
	result := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("s.no_wa IN ? AND s.school_id = ?", variants, schoolID).
		First(&siswa)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &siswa, nil
}

func GetStudentByNIS(nis, schoolID string) (*Siswa, error) {
	var siswa Siswa
	result := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("s.nis = ? AND s.school_id = ?", nis, schoolID).
		First(&siswa)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &siswa, nil
}

func IsBotEnabled(schoolID string) bool {
	var result struct {
		WaEnabled bool `gorm:"column:wa_enabled"`
		BotEnabled bool `gorm:"column:bot_enabled"`
	}
	err := config.DB.Table("schools").
		Select("wa_enabled, bot_enabled").
		Where("id = ?", schoolID).
		First(&result).Error
	if err != nil {
		// School not found or DB error → fail open agar bot tetap jalan
		return true
	}
	return result.WaEnabled && result.BotEnabled
}

func HasTeacherBotAccess(teacherID uint) bool {
	var guru Guru
	result := config.DB.Table("guru").Where("id = ?", teacherID).First(&guru)
	if result.Error != nil {
		return true // fail open
	}
	return guru.BotAccess
}

func GetSchoolName(schoolID string) string {
	var name string
	config.DB.Table("schools").Select("name").Where("id = ?", schoolID).Scan(&name)
	if name == "" {
		return "Sekolah Anda"
	}
	return name
}

func GetTodayAttendance(studentID uint) (*Attendance, error) {
	var att Attendance
	result := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, today()).
		First(&att)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &att, nil
}

func GetAttendanceRecap(studentID uint, period string) (*AttendanceStats, error) {
	loc := GetJakartaLocation()
	now := time.Now().In(loc)

	var startDate, endDate string
	switch period {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")
		endDate = now.AddDate(0, 0, 7-weekday).Format("2006-01-02")
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).Format("2006-01-02")
		endDate = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, loc).Format("2006-01-02")
	default: // today
		startDate = now.Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	}

	var records []Attendance
	result := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal BETWEEN ? AND ?", studentID, startDate, endDate).
		Order("tanggal DESC").
		Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}

	stats := &AttendanceStats{Total: len(records), Records: records}
	for _, r := range records {
		switch r.Status {
		case "H":
			stats.Hadir++
		case "I":
			stats.Izin++
		case "S":
			stats.Sakit++
		case "A":
			stats.Alpha++
		case "B":
			stats.Bolos++
		}
	}
	return stats, nil
}

func SearchStudentsByName(searchTerm, schoolID string) ([]Siswa, error) {
	var students []Siswa
	result := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("s.nama LIKE ? AND s.school_id = ?", "%"+searchTerm+"%", schoolID).
		Order("s.nama").
		Limit(10).
		Find(&students)
	return students, result.Error
}

func SearchStudentsForRegistration(searchTerm, schoolID string) ([]Siswa, error) {
	var students []Siswa
	result := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("(s.nama LIKE ? OR s.nis = ?) AND s.school_id = ?", "%"+searchTerm+"%", searchTerm, schoolID).
		Order("s.nama").
		Limit(10).
		Find(&students)
	return students, result.Error
}

func UpdateStudentPhone(studentID uint, phone string, isOrtu bool) error {
	field := "no_wa"
	if isOrtu {
		field = "wa_ortu"
	}
	return config.DB.Table("siswa").Where("id = ?", studentID).Update(field, phone).Error
}

func SearchStudentsWithAttendanceToday(searchTerm, schoolID string) ([]Siswa, error) {
	var students []Siswa
	result := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Joins("INNER JOIN attendance a ON s.id = a.student_id AND a.tanggal = ?", today()).
		Where("s.nama LIKE ? AND s.school_id = ?", "%"+searchTerm+"%", schoolID).
		Order("s.nama").
		Limit(10).
		Find(&students)
	return students, result.Error
}

func GetStudentAttendanceToday(studentID uint) (*Attendance, error) {
	var att Attendance
	result := config.DB.Table("attendance a").
		Select("a.*, s.nama, k.nama_kelas").
		Joins("INNER JOIN siswa s ON a.student_id = s.id").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("a.student_id = ? AND a.tanggal = ?", studentID, today()).
		First(&att)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &att, nil
}

// GetTodayScheduledJamPulang returns the scheduled jam_pulang for a school today (e.g. "12:00:00").
func GetTodayScheduledJamPulang(schoolID interface{}) string {
	loc := GetJakartaLocation()
	now := time.Now().In(loc)
	weekday := int(now.Weekday())
	indexHari := weekday
	if weekday == 0 {
		indexHari = 7
	}

	var result struct {
		JamPulang string `gorm:"column:jam_pulang"`
	}

	err := config.DB.Table("jadwal").
		Select("jam_pulang").
		Where("school_id = ? AND index_hari = ? AND is_active = 1", schoolID, indexHari).
		First(&result).Error

	if err == nil && result.JamPulang != "" {
		return result.JamPulang
	}

	// Fallback if not found for specific school_id
	config.DB.Table("jadwal").
		Select("jam_pulang").
		Where("school_id IS NULL AND index_hari = ? AND is_active = 1", indexHari).
		First(&result)

	return result.JamPulang
}

func calculateTotalSeconds(masukStr, pulangStr string) int {
	layouts := []string{"15:04:05", "15:04"}
	var tMasuk, tPulang time.Time
	for _, l := range layouts {
		if t, err := time.Parse(l, masukStr); err == nil {
			tMasuk = t
			break
		}
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, pulangStr); err == nil {
			tPulang = t
			break
		}
	}
	if !tMasuk.IsZero() && !tPulang.IsZero() && tPulang.After(tMasuk) {
		return int(tPulang.Sub(tMasuk).Seconds())
	}
	return 0
}

func CreateManualAttendance(studentID uint, status, teacherName, keterangan, schoolID string, teacherID uint) (string, error) {
	todayStr := today()
	nowStr := nowTime()
	finalKet := keterangan
	if finalKet == "" {
		finalKet = fmt.Sprintf("Absen manual oleh %s", teacherName)
	}

	var scheduledJamPulang string
	var totalSecs int

	if status == "H" {
		targetSchoolID := schoolID
		var studentSchoolID uint
		if err := config.DB.Table("siswa").Select("school_id").Where("id = ?", studentID).Scan(&studentSchoolID).Error; err == nil && studentSchoolID != 0 {
			targetSchoolID = fmt.Sprintf("%d", studentSchoolID)
		}

		scheduledJamPulang = GetTodayScheduledJamPulang(targetSchoolID)
		if scheduledJamPulang != "" {
			totalSecs = calculateTotalSeconds(nowStr, scheduledJamPulang)
		}
	}

	var existing Attendance
	err := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		First(&existing).Error

	if err == nil {
		// update
		if status == "H" {
			updates := map[string]interface{}{
				"jam_masuk":   nowStr,
				"status":      status,
				"keterangan":  finalKet,
				"updated_at":  time.Now(),
			}
			if scheduledJamPulang != "" {
				updates["jam_pulang"] = scheduledJamPulang
				updates["total_seconds"] = totalSecs
			}
			if teacherID != 0 {
				updates["checked_in_by_teacher_id"] = teacherID
			}
			config.DB.Table("attendance").
				Where("student_id = ? AND tanggal = ?", studentID, todayStr).
				Updates(updates)
		} else {
			config.DB.Table("attendance").
				Where("student_id = ? AND tanggal = ?", studentID, todayStr).
				Updates(map[string]interface{}{"status": status, "keterangan": finalKet, "updated_at": time.Now()})
		}
	} else {
		// insert
		if status == "H" {
			if scheduledJamPulang != "" {
				if teacherID != 0 {
					config.DB.Exec(
						"INSERT INTO attendance (student_id, tanggal, jam_masuk, jam_pulang, total_seconds, status, keterangan, checked_in_by_teacher_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
						studentID, todayStr, nowStr, scheduledJamPulang, totalSecs, status, finalKet, teacherID,
					)
				} else {
					config.DB.Exec(
						"INSERT INTO attendance (student_id, tanggal, jam_masuk, jam_pulang, total_seconds, status, keterangan, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
						studentID, todayStr, nowStr, scheduledJamPulang, totalSecs, status, finalKet,
					)
				}
			} else {
				if teacherID != 0 {
					config.DB.Exec(
						"INSERT INTO attendance (student_id, tanggal, jam_masuk, status, keterangan, checked_in_by_teacher_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())",
						studentID, todayStr, nowStr, status, finalKet, teacherID,
					)
				} else {
					config.DB.Exec(
						"INSERT INTO attendance (student_id, tanggal, jam_masuk, status, keterangan, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
						studentID, todayStr, nowStr, status, finalKet,
					)
				}
			}
		} else {
			config.DB.Exec(
				"INSERT INTO attendance (student_id, tanggal, status, keterangan, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())",
				studentID, todayStr, status, finalKet,
			)
		}
	}

	// Notify student & parent in background
	go NotifyStudentAndParentAttendance(studentID, status, finalKet, teacherName, schoolID, scheduledJamPulang)
	return scheduledJamPulang, nil
}

func UpdateAttendanceStatus(studentID uint, status, teacherName, schoolID string) error {
	todayStr := today()
	nowStr := nowTime()
	keterangan := fmt.Sprintf("Status diubah oleh %s", teacherName)

	updates := map[string]interface{}{
		"status":     status,
		"keterangan": keterangan,
		"updated_at": time.Now(),
	}

	if status == "H" {
		var existing Attendance
		if err := config.DB.Table("attendance").Where("student_id = ? AND tanggal = ?", studentID, todayStr).First(&existing).Error; err == nil {
			if existing.JamMasuk == "" {
				updates["jam_masuk"] = nowStr
			}
			if existing.JamPulang == "" {
				targetSchoolID := schoolID
				var studentSchoolID uint
				if err := config.DB.Table("siswa").Select("school_id").Where("id = ?", studentID).Scan(&studentSchoolID).Error; err == nil && studentSchoolID != 0 {
					targetSchoolID = fmt.Sprintf("%d", studentSchoolID)
				}
				scheduledJamPulang := GetTodayScheduledJamPulang(targetSchoolID)
				if scheduledJamPulang != "" {
					updates["jam_pulang"] = scheduledJamPulang
					masuk := existing.JamMasuk
					if masuk == "" {
						masuk = nowStr
					}
					updates["total_seconds"] = calculateTotalSeconds(masuk, scheduledJamPulang)
				}
			}
		}
	}

	config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		Updates(updates)
	go NotifyStudentAndParentEdit(studentID, status, keterangan, teacherName, schoolID)
	return nil
}

func UpdateAttendanceKeterangan(studentID uint, keterangan, teacherName, schoolID string) error {
	todayStr := today()
	finalKet := fmt.Sprintf("%s (diubah oleh %s)", keterangan, teacherName)
	config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		Updates(map[string]interface{}{"keterangan": finalKet, "updated_at": time.Now()})
	return nil
}

func DeleteAttendanceToday(studentID uint) error {
	config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, today()).
		Delete(nil)
	return nil
}

func QuickCheckin(studentID, teacherID uint, teacherName, schoolID string) error {
	todayStr := today()
	nowStr := nowTime()
	keterangan := fmt.Sprintf("Absen masuk oleh %s", teacherName)

	var existing Attendance
	err := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		First(&existing).Error

	if err == nil {
		config.DB.Table("attendance").
			Where("student_id = ? AND tanggal = ?", studentID, todayStr).
			Updates(map[string]interface{}{
				"jam_masuk":                 nowStr,
				"status":                    "H",
				"keterangan":                keterangan,
				"checked_in_by_teacher_id":  teacherID,
				"updated_at":                time.Now(),
			})
	} else {
		config.DB.Exec(
			"INSERT INTO attendance (student_id, tanggal, jam_masuk, status, keterangan, checked_in_by_teacher_id, created_at, updated_at) VALUES (?, ?, ?, 'H', ?, ?, NOW(), NOW())",
			studentID, todayStr, nowStr, keterangan, teacherID,
		)
	}

	go NotifyStudentAndParentCheckin(studentID, teacherName, schoolID)
	return nil
}

type CheckoutResult struct {
	Success   bool
	JamMasuk  string
}

func QuickCheckout(studentID uint, teacherName, schoolID string) CheckoutResult {
	todayStr := today()
	nowStr := nowTime()

	var existing Attendance
	err := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		First(&existing).Error

	if err != nil {
		return CheckoutResult{Success: false}
	}

	config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		Updates(map[string]interface{}{"jam_pulang": nowStr, "status": "H", "updated_at": time.Now()})

	go NotifyStudentAndParentCheckout(studentID, existing.JamMasuk, teacherName, schoolID)
	return CheckoutResult{Success: true, JamMasuk: existing.JamMasuk}
}

func SearchStudentContact(searchTerm, schoolID string) ([]Siswa, error) {
	var students []Siswa
	result := config.DB.Table("siswa s").
		Select("s.id, s.nama, s.nis, s.no_wa, s.wa_ortu, s.tgl_lahir, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("(s.nama LIKE ? OR s.nis LIKE ?) AND s.school_id = ?", "%"+searchTerm+"%", "%"+searchTerm+"%", schoolID).
		Order("s.nama").
		Limit(5).
		Find(&students)
	return students, result.Error
}

// ─── Notification Helpers ─────────────────────────────────────────────────────

func getStudentWithClass(studentID uint) (*Siswa, error) {
	var s Siswa
	err := config.DB.Table("siswa s").
		Select("s.*, k.nama_kelas").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("s.id = ?", studentID).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func IsWithinTwoDays(lastSeen *time.Time) bool {
	if lastSeen == nil {
		return false
	}
	limit := time.Now().Add(-48 * time.Hour)
	return lastSeen.After(limit)
}

func NotifyStudentAndParentAttendance(studentID uint, status, keterangan, teacherName, schoolID string, jamPulang ...string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	pulang := ""
	if len(jamPulang) > 0 {
		pulang = jamPulang[0]
	}
	if s.NoWa != "" {
		if IsWithinTwoDays(s.LastSeenSiswa) {
			msg := GenerateStudentAttendanceNotification(s.Nama, status, keterangan, teacherName, pulang)
			SendMessage(s.NoWa, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip student notification to %s: No interaction within 2 days\n", s.NoWa)
		}
	}
	if s.WaOrtu != "" {
		if IsWithinTwoDays(s.LastSeenOrtu) {
			msg := GenerateParentAttendanceNotification(s.Nama, s.NamaKelas, status, keterangan, teacherName, pulang)
			SendMessage(s.WaOrtu, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip parent notification to %s: No interaction within 2 days\n", s.WaOrtu)
		}
	}
}

func NotifyStudentAndParentCheckin(studentID uint, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	if s.NoWa != "" {
		if IsWithinTwoDays(s.LastSeenSiswa) {
			msg := GenerateStudentCheckinNotification(s.Nama, teacherName)
			SendMessage(s.NoWa, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip student checkin notification to %s: No interaction within 2 days\n", s.NoWa)
		}
	}
	if s.WaOrtu != "" {
		if IsWithinTwoDays(s.LastSeenOrtu) {
			msg := GenerateParentCheckinNotification(s.Nama, s.NamaKelas, teacherName)
			SendMessage(s.WaOrtu, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip parent checkin notification to %s: No interaction within 2 days\n", s.WaOrtu)
		}
	}
}

func NotifyStudentAndParentCheckout(studentID uint, jamMasuk, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	if s.NoWa != "" {
		if IsWithinTwoDays(s.LastSeenSiswa) {
			msg := GenerateStudentCheckoutNotification(s.Nama, jamMasuk, teacherName)
			SendMessage(s.NoWa, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip student checkout notification to %s: No interaction within 2 days\n", s.NoWa)
		}
	}
	if s.WaOrtu != "" {
		if IsWithinTwoDays(s.LastSeenOrtu) {
			msg := GenerateParentCheckoutNotification(s.Nama, s.NamaKelas, jamMasuk, teacherName)
			SendMessage(s.WaOrtu, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip parent checkout notification to %s: No interaction within 2 days\n", s.WaOrtu)
		}
	}
}

func NotifyStudentAndParentEdit(studentID uint, status, keterangan, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	statusLabels := map[string]string{"H": "Hadir", "I": "Izin", "S": "Sakit", "A": "Alpha"}
	if s.NoWa != "" {
		if IsWithinTwoDays(s.LastSeenSiswa) {
			msg := fmt.Sprintf("✏️ *Absensi Anda Diperbarui*\n\nStatus baru: *%s*\nOleh: %s", statusLabels[status], teacherName)
			SendMessage(s.NoWa, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip student edit notification to %s: No interaction within 2 days\n", s.NoWa)
		}
	}
	if s.WaOrtu != "" {
		if IsWithinTwoDays(s.LastSeenOrtu) {
			msg := GenerateParentEditNotification(s.Nama, s.NamaKelas, status, keterangan, teacherName)
			SendMessage(s.WaOrtu, msg, schoolID)
		} else {
			fmt.Printf("⚠️ Skip parent edit notification to %s: No interaction within 2 days\n", s.WaOrtu)
		}
	}
}

// ─── Last Seen Migrator & Updates ─────────────────────────────────────────────

func MigrateLastSeen() {
	var count int64

	// Check and add last_seen_siswa to siswa table
	config.DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'siswa' AND COLUMN_NAME = 'last_seen_siswa'").Scan(&count)
	if count == 0 {
		fmt.Println("Migrating: Adding last_seen_siswa to siswa table")
		if err := config.DB.Exec("ALTER TABLE `siswa` ADD COLUMN `last_seen_siswa` DATETIME NULL").Error; err != nil {
			fmt.Printf("Warning: Failed to add last_seen_siswa: %v\n", err)
		}
	}

	// Check and add last_seen_ortu to siswa table
	config.DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'siswa' AND COLUMN_NAME = 'last_seen_ortu'").Scan(&count)
	if count == 0 {
		fmt.Println("Migrating: Adding last_seen_ortu to siswa table")
		if err := config.DB.Exec("ALTER TABLE `siswa` ADD COLUMN `last_seen_ortu` DATETIME NULL").Error; err != nil {
			fmt.Printf("Warning: Failed to add last_seen_ortu: %v\n", err)
		}
	}

	// Check and add last_seen to guru table
	config.DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guru' AND COLUMN_NAME = 'last_seen'").Scan(&count)
	if count == 0 {
		fmt.Println("Migrating: Adding last_seen to guru table")
		if err := config.DB.Exec("ALTER TABLE `guru` ADD COLUMN `last_seen` DATETIME NULL").Error; err != nil {
			fmt.Printf("Warning: Failed to add last_seen to guru: %v\n", err)
		}
	}
}

func UpdateLastSeen(phone string) {
	if phone == "" {
		return
	}
	variants := phoneVariants(phone)
	
	loc := GetJakartaLocation()
	now := time.Now().In(loc)

	// Update siswa last_seen_siswa
	if err := config.DB.Table("siswa").Where("no_wa IN ?", variants).Update("last_seen_siswa", now).Error; err != nil {
		fmt.Printf("Error updating last_seen_siswa for phone %s: %v\n", phone, err)
	}

	// Update siswa last_seen_ortu
	if err := config.DB.Table("siswa").Where("wa_ortu IN ?", variants).Update("last_seen_ortu", now).Error; err != nil {
		fmt.Printf("Error updating last_seen_ortu for phone %s: %v\n", phone, err)
	}

	// Update guru last_seen
	if err := config.DB.Table("guru").Where("no_wa IN ?", variants).Update("last_seen", now).Error; err != nil {
		fmt.Printf("Error updating last_seen for guru phone %s: %v\n", phone, err)
	}
}
