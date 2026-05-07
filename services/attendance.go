package services

import (
	"fmt"
	"strings"
	"time"

	"bot_wa/config"

	"gorm.io/gorm"
)

// ─── Models ───────────────────────────────────────────────────────────────────

type Siswa struct {
	ID       uint   `gorm:"column:id"`
	Nama     string `gorm:"column:nama"`
	NIS      string `gorm:"column:nis"`
	NoWa     string `gorm:"column:no_wa"`
	WaOrtu   string `gorm:"column:wa_ortu"`
	TglLahir string `gorm:"column:tgl_lahir"`
	KelasID  uint   `gorm:"column:kelas_id"`
	SchoolID uint   `gorm:"column:school_id"`
	NamaKelas string `gorm:"column:nama_kelas"`
}

type Guru struct {
	ID        uint   `gorm:"column:id"`
	Nama      string `gorm:"column:nama"`
	NoWa      string `gorm:"column:no_wa"`
	SchoolID  uint   `gorm:"column:school_id"`
	BotAccess bool   `gorm:"column:bot_access"`
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
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("2006-01-02")
}

func nowTime() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
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
	loc, _ := time.LoadLocation("Asia/Jakarta")
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

func CreateManualAttendance(studentID uint, status, teacherName, keterangan, schoolID string, teacherID uint) error {
	todayStr := today()
	nowStr := nowTime()
	finalKet := keterangan
	if finalKet == "" {
		finalKet = fmt.Sprintf("Absen manual oleh %s", teacherName)
	}

	var existing Attendance
	err := config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		First(&existing).Error

	if err == nil {
		// update
		if status == "H" {
			config.DB.Table("attendance").
				Where("student_id = ? AND tanggal = ?", studentID, todayStr).
				Updates(map[string]interface{}{"jam_masuk": nowStr, "status": status, "keterangan": finalKet, "updated_at": time.Now()})
		} else {
			config.DB.Table("attendance").
				Where("student_id = ? AND tanggal = ?", studentID, todayStr).
				Updates(map[string]interface{}{"status": status, "keterangan": finalKet, "updated_at": time.Now()})
		}
	} else {
		// insert
		if status == "H" {
			config.DB.Exec(
				"INSERT INTO attendance (student_id, tanggal, jam_masuk, status, keterangan, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
				studentID, todayStr, nowStr, status, finalKet,
			)
		} else {
			config.DB.Exec(
				"INSERT INTO attendance (student_id, tanggal, status, keterangan, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())",
				studentID, todayStr, status, finalKet,
			)
		}
	}

	// Notify student & parent in background
	go NotifyStudentAndParentAttendance(studentID, status, finalKet, teacherName, schoolID)
	return nil
}

func UpdateAttendanceStatus(studentID uint, status, teacherName, schoolID string) error {
	todayStr := today()
	keterangan := fmt.Sprintf("Status diubah oleh %s", teacherName)
	config.DB.Table("attendance").
		Where("student_id = ? AND tanggal = ?", studentID, todayStr).
		Updates(map[string]interface{}{"status": status, "keterangan": keterangan, "updated_at": time.Now()})
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
		Select("s.id, s.nama, s.nis, s.no_wa, s.wa_ortu, k.nama_kelas").
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

func NotifyStudentAndParentAttendance(studentID uint, status, keterangan, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	if s.NoWa != "" {
		msg := GenerateStudentAttendanceNotification(s.Nama, status, keterangan, teacherName)
		SendMessage(s.NoWa, msg, schoolID)
	}
	if s.WaOrtu != "" {
		msg := GenerateParentAttendanceNotification(s.Nama, s.NamaKelas, status, keterangan, teacherName)
		SendMessage(s.WaOrtu, msg, schoolID)
	}
}

func NotifyStudentAndParentCheckin(studentID uint, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	if s.NoWa != "" {
		msg := GenerateStudentCheckinNotification(s.Nama, teacherName)
		SendMessage(s.NoWa, msg, schoolID)
	}
	if s.WaOrtu != "" {
		msg := GenerateParentCheckinNotification(s.Nama, s.NamaKelas, teacherName)
		SendMessage(s.WaOrtu, msg, schoolID)
	}
}

func NotifyStudentAndParentCheckout(studentID uint, jamMasuk, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	if s.NoWa != "" {
		msg := GenerateStudentCheckoutNotification(s.Nama, jamMasuk, teacherName)
		SendMessage(s.NoWa, msg, schoolID)
	}
	if s.WaOrtu != "" {
		msg := GenerateParentCheckoutNotification(s.Nama, s.NamaKelas, jamMasuk, teacherName)
		SendMessage(s.WaOrtu, msg, schoolID)
	}
}

func NotifyStudentAndParentEdit(studentID uint, status, keterangan, teacherName, schoolID string) {
	s, err := getStudentWithClass(studentID)
	if err != nil || s == nil {
		return
	}
	statusLabels := map[string]string{"H": "Hadir", "I": "Izin", "S": "Sakit", "A": "Alpha"}
	if s.NoWa != "" {
		msg := fmt.Sprintf("✏️ *Absensi Anda Diperbarui*\n\nStatus baru: *%s*\nOleh: %s", statusLabels[status], teacherName)
		SendMessage(s.NoWa, msg, schoolID)
	}
	if s.WaOrtu != "" {
		msg := GenerateParentEditNotification(s.Nama, s.NamaKelas, status, keterangan, teacherName)
		SendMessage(s.WaOrtu, msg, schoolID)
	}
}
