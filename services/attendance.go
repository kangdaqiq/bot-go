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
	ID             uint       `gorm:"column:id"`
	UserID         *uint      `gorm:"column:user_id"`
	Nama           string     `gorm:"column:nama"`
	NoWa           string     `gorm:"column:no_wa"`
	SchoolID       uint       `gorm:"column:school_id"`
	BotAccess      bool       `gorm:"column:bot_access"`
	IsGlobalReport bool       `gorm:"column:is_global_report"`
	LastSeen       *time.Time `gorm:"column:last_seen"`
}

type StudentLeave struct {
	ID             uint       `gorm:"column:id"`
	SchoolID       uint       `gorm:"column:school_id"`
	StudentID      uint       `gorm:"column:student_id"`
	Code           string     `gorm:"column:code"`
	Jenis          string     `gorm:"column:jenis"`
	TanggalMulai   string     `gorm:"column:tanggal_mulai"`
	TanggalSelesai string     `gorm:"column:tanggal_selesai"`
	Keterangan     string     `gorm:"column:keterangan"`
	BuktiFoto      string     `gorm:"column:bukti_foto"`
	Pengaju        string     `gorm:"column:pengaju"`
	NamaPengaju    string     `gorm:"column:nama_pengaju"`
	NoWaPengaju    string     `gorm:"column:no_wa_pengaju"`
	Status         string     `gorm:"column:status"`
	ApprovedBy     *uint      `gorm:"column:approved_by"`
	ApprovedAt     *time.Time `gorm:"column:approved_at"`
	RejectedReason string     `gorm:"column:rejected_reason"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`

	// Joined fields
	NamaSiswa    string `gorm:"column:nama_siswa"`
	NIS          string `gorm:"column:nis"`
	NoWaSiswa    string `gorm:"column:no_wa_siswa"`
	WaOrtu       string `gorm:"column:wa_ortu"`
	NamaKelas    string `gorm:"column:nama_kelas"`
	KelasID      uint   `gorm:"column:kelas_id"`
	WaliKelasID  *uint  `gorm:"column:wali_kelas_id"`
	WaliKelas2ID *uint  `gorm:"column:wali_kelas_2_id"`
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

	if cleaned == "" {
		return ""
	}

	// Prefix 00 (international dialing prefix seperti 0060...) dihilangkan
	if strings.HasPrefix(cleaned, "00") {
		cleaned = cleaned[2:]
	}

	// Format nomor lokal Indonesia diawali 0 (misal: 0812...) -> ubah 0 jadi 62
	if strings.HasPrefix(cleaned, "0") {
		return "62" + cleaned[1:]
	}

	// Format nomor Indonesia tanpa awalan 0 tapi diawali 8 (misal: 812...) -> tambahkan 62
	if strings.HasPrefix(cleaned, "8") {
		return "62" + cleaned
	}

	// Jika sudah diawali kode negara (baik 62 untuk Indonesia atau kode negara lain seperti 60, 966, 1, dll)
	return cleaned
}

func phoneVariants(phone string) []string {
	norm := NormalizePhone(phone)
	if norm == "" {
		return nil
	}

	variantsMap := make(map[string]bool)
	variantsMap[norm] = true
	variantsMap["+"+norm] = true

	if strings.HasPrefix(norm, "62") {
		variantsMap["0"+norm[2:]] = true
		variantsMap[norm[2:]] = true
		variantsMap["+62"+norm[2:]] = true
	} else {
		variantsMap["00"+norm] = true
	}

	var variants []string
	for v := range variantsMap {
		variants = append(variants, v)
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

// ─── Student Leave Approval & Management ─────────────────────────────────────

func parseDateFlexible(dateStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if len(trimmed) >= 10 {
		dateOnly := trimmed[:10]
		if t, err := time.Parse("2006-01-02", dateOnly); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", dateStr)
}

func FormatDateRangeIndo(start, end string) string {
	tStart, errStart := parseDateFlexible(start)
	tEnd, errEnd := parseDateFlexible(end)
	if errStart != nil || errEnd != nil {
		if start == end || end == "" {
			return start
		}
		return fmt.Sprintf("%s s/d %s", start, end)
	}
	fStart := tStart.Format("02/01/2006")
	fEnd := tEnd.Format("02/01/2006")
	if fStart == fEnd {
		return fStart
	}
	return fmt.Sprintf("%s s/d %s", fStart, fEnd)
}

func syncLeaveAttendance(leave *StudentLeave, teacherName string) error {
	startDate, errStart := parseDateFlexible(leave.TanggalMulai)
	endDate, errEnd := parseDateFlexible(leave.TanggalSelesai)
	if errStart != nil || errEnd != nil {
		fmt.Printf("❌ Error parsing leave dates: start=%s (err: %v), end=%s (err: %v)\n", leave.TanggalMulai, errStart, leave.TanggalSelesai, errEnd)
		return fmt.Errorf("format tanggal izin tidak valid: %s s/d %s", leave.TanggalMulai, leave.TanggalSelesai)
	}

	statusCode := "I"
	if strings.ToLower(leave.Jenis) == "sakit" {
		statusCode = "S"
	}
	keteranganAtt := fmt.Sprintf("[Pengajuan %s Disetujui %s] %s", strings.Title(leave.Jenis), teacherName, leave.Keterangan)

	now := time.Now()
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		curDate := d.Format("2006-01-02")
		var existingAtt Attendance
		errFind := config.DB.Table("attendance").Where("student_id = ? AND tanggal = ?", leave.StudentID, curDate).First(&existingAtt).Error
		if errFind == nil && existingAtt.ID > 0 {
			errUpd := config.DB.Table("attendance").Where("id = ?", existingAtt.ID).Updates(map[string]interface{}{
				"status":        statusCode,
				"keterangan":    keteranganAtt,
				"is_auto_alpha": 0,
				"updated_at":    now,
			}).Error
			if errUpd != nil {
				fmt.Printf("❌ Failed to update attendance for student %d on %s: %v\n", leave.StudentID, curDate, errUpd)
			} else {
				fmt.Printf("✅ Attendance updated: student %d on %s status -> %s\n", leave.StudentID, curDate, statusCode)
			}
		} else {
			errIns := config.DB.Exec(
				"INSERT INTO attendance (student_id, tanggal, status, keterangan, total_seconds, is_auto_alpha, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 0, NOW(), NOW())",
				leave.StudentID, curDate, statusCode, keteranganAtt,
			).Error
			if errIns != nil {
				fmt.Printf("❌ Failed to insert attendance for student %d on %s: %v\n", leave.StudentID, curDate, errIns)
			} else {
				fmt.Printf("✅ Attendance inserted: student %d on %s status -> %s\n", leave.StudentID, curDate, statusCode)
			}
		}
	}
	return nil
}

func getUniquePhones(phones ...string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, p := range phones {
		norm := NormalizePhone(p)
		if norm != "" && !seen[norm] {
			seen[norm] = true
			result = append(result, norm)
		}
	}
	return result
}

func GetStudentLeaveByCode(code string, schoolID string) (*StudentLeave, error) {
	var leave StudentLeave
	query := config.DB.Table("student_leaves l").
		Select("l.*, s.nama as nama_siswa, s.nis, s.no_wa as no_wa_siswa, s.wa_ortu, s.kelas_id, k.nama_kelas, k.wali_kelas_id, k.wali_kelas_2_id").
		Joins("INNER JOIN siswa s ON l.student_id = s.id").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("UPPER(l.code) = UPPER(?)", code)

	if schoolID != "" && schoolID != "1" {
		query = query.Where("l.school_id = ?", schoolID)
	}

	result := query.First(&leave)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &leave, nil
}

func ApproveStudentLeaveByTeacher(code string, teacher *Guru, schoolID string) string {
	if code == "" {
		return "⚠️ *Kode Izin Kosong*\n\nSilakan tentukan kode izin yang ingin disetujui.\nContoh: `acc IZIN-ABC123`\n\nKetik `izin pending` untuk melihat daftar pengajuan."
	}

	leave, err := GetStudentLeaveByCode(code, schoolID)
	if err != nil {
		return fmt.Sprintf("❌ *Gagal Mengambil Data Izin*\n\nTerjadi kesalahan database: %v", err)
	}
	if leave == nil {
		return fmt.Sprintf("⚠️ *Pengajuan Izin Tidak Ditemukan*\n\nKode izin *%s* tidak ditemukan pada sistem sekolah Anda.\n\nPastikan kode yang dimasukkan sudah benar atau ketik `izin pending`.", code)
	}

	// Cek Hak Akses (Wali Kelas 1 / Wali Kelas 2 atau Global Report)
	isWali := (leave.WaliKelasID != nil && *leave.WaliKelasID == teacher.ID) ||
		(leave.WaliKelas2ID != nil && *leave.WaliKelas2ID == teacher.ID)

	if !isWali && !teacher.IsGlobalReport {
		return fmt.Sprintf("⚠️ *Akses Ditolak*\n\nBapak/Ibu *%s* bukan Wali Kelas dari siswa *%s* (%s), sehingga tidak memiliki hak akses untuk menyetujui pengajuan izin ini.", teacher.Nama, leave.NamaSiswa, leave.NamaKelas)
	}

	// Cek Status Pengajuan
	if leave.Status != "pending" {
		if leave.Status == "approved" {
			// Sinkronkan ulang ke tabel attendance jika sebelumnya belum tersinkron
			_ = syncLeaveAttendance(leave, teacher.Nama)
			dateRange := FormatDateRangeIndo(leave.TanggalMulai, leave.TanggalSelesai)
			return fmt.Sprintf("✅ *Absensi Berhasil Disinkronkan!*\n\nPengajuan izin kode *%s* an. *%s* (%s) sudah berstatus disetujui sebelumnya, dan data absensi tanggal *%s* kini telah dipastikan tersinkron sebagai *%s*.", leave.Code, leave.NamaSiswa, leave.NamaKelas, dateRange, strings.Title(leave.Jenis))
		}
		if leave.Status == "rejected" {
			return fmt.Sprintf("ℹ️ *Pengajuan Sudah Ditolak*\n\nPengajuan izin kode *%s* an. *%s* (%s) sudah berstatus *Ditolak* sebelumnya.", leave.Code, leave.NamaSiswa, leave.NamaKelas)
		}
		return fmt.Sprintf("ℹ️ *Status Pengajuan: %s*\n\nHanya pengajuan berstatus pending yang dapat disetujui.", leave.Status)
	}

	now := time.Now()
	var approverID *uint = teacher.UserID
	if approverID == nil || *approverID == 0 {
		var uID uint
		if err := config.DB.Table("users").Select("id").Where("id = ? OR username = ?", teacher.ID, teacher.Nama).Scan(&uID).Error; err == nil && uID > 0 {
			approverID = &uID
		}
	}

	updates := map[string]interface{}{
		"status":          "approved",
		"approved_at":     now,
		"rejected_reason": nil,
		"updated_at":      now,
	}
	if approverID != nil && *approverID > 0 {
		updates["approved_by"] = *approverID
	}

	if err := config.DB.Table("student_leaves").Where("id = ?", leave.ID).Updates(updates).Error; err != nil {
		return fmt.Sprintf("❌ *Gagal Menyetujui Izin*\n\nTerjadi kesalahan database: %v", err)
	}

	// Sinkronisasi ke tabel attendance
	if errSync := syncLeaveAttendance(leave, teacher.Nama); errSync != nil {
		fmt.Printf("⚠️ Warning: syncLeaveAttendance error: %v\n", errSync)
	}

	// Kirim notifikasi WhatsApp ke Orang Tua / Siswa secara background
	go func() {
		targetSchoolID := schoolID
		if targetSchoolID == "" || targetSchoolID == "1" {
			targetSchoolID = fmt.Sprintf("%d", leave.SchoolID)
		}
		notifMsg := GenerateParentLeaveApprovedNotification(
			leave.NamaSiswa,
			leave.Jenis,
			leave.TanggalMulai,
			leave.TanggalSelesai,
			teacher.Nama,
		)

		phones := getUniquePhones(leave.NoWaPengaju, leave.WaOrtu, leave.NoWaSiswa)
		for _, p := range phones {
			SendMessage(p, notifMsg, targetSchoolID)
		}
	}()

	dateRange := FormatDateRangeIndo(leave.TanggalMulai, leave.TanggalSelesai)
	return fmt.Sprintf("✅ *Pengajuan Izin Disetujui!*\n\nPermohonan *%s* untuk siswa:\n👤 Nama: *%s*\n🏫 Kelas: *%s*\n📅 Tanggal: *%s*\n\nData absensi siswa telah otomatis diperbarui dan notifikasi persetujuan telah diteruskan ke nomor orang tua/siswa.", strings.Title(leave.Jenis), leave.NamaSiswa, leave.NamaKelas, dateRange)
}

func RejectStudentLeaveByTeacher(code, reason string, teacher *Guru, schoolID string) string {
	if code == "" {
		return "⚠️ *Kode Izin Kosong*\n\nSilakan tentukan kode izin yang ingin ditolak.\nContoh: `tolak IZIN-ABC123 Alasan tidak jelas`\n\nKetik `izin pending` untuk melihat daftar pengajuan."
	}

	leave, err := GetStudentLeaveByCode(code, schoolID)
	if err != nil {
		return fmt.Sprintf("❌ *Gagal Mengambil Data Izin*\n\nTerjadi kesalahan database: %v", err)
	}
	if leave == nil {
		return fmt.Sprintf("⚠️ *Pengajuan Izin Tidak Ditemukan*\n\nKode izin *%s* tidak ditemukan pada sistem sekolah Anda.\n\nPastikan kode yang dimasukkan sudah benar atau ketik `izin pending`.", code)
	}

	// Cek Hak Akses (Wali Kelas 1 / Wali Kelas 2 atau Global Report)
	isWali := (leave.WaliKelasID != nil && *leave.WaliKelasID == teacher.ID) ||
		(leave.WaliKelas2ID != nil && *leave.WaliKelas2ID == teacher.ID)

	if !isWali && !teacher.IsGlobalReport {
		return fmt.Sprintf("⚠️ *Akses Ditolak*\n\nBapak/Ibu *%s* bukan Wali Kelas dari siswa *%s* (%s), sehingga tidak memiliki hak akses untuk menolak pengajuan izin ini.", teacher.Nama, leave.NamaSiswa, leave.NamaKelas)
	}

	// Cek Status Pengajuan
	if leave.Status != "pending" {
		if leave.Status == "approved" {
			return fmt.Sprintf("ℹ️ *Pengajuan Sudah Disetujui*\n\nPengajuan izin kode *%s* an. *%s* (%s) sudah berstatus *Disetujui* sebelumnya.", leave.Code, leave.NamaSiswa, leave.NamaKelas)
		}
		if leave.Status == "rejected" {
			return fmt.Sprintf("ℹ️ *Pengajuan Sudah Ditolak*\n\nPengajuan izin kode *%s* an. *%s* (%s) sudah berstatus *Ditolak* sebelumnya.", leave.Code, leave.NamaSiswa, leave.NamaKelas)
		}
		return fmt.Sprintf("ℹ️ *Status Pengajuan: %s*\n\nHanya pengajuan berstatus pending yang dapat diproses.", leave.Status)
	}

	now := time.Now()
	finalReason := strings.TrimSpace(reason)
	if finalReason == "" {
		finalReason = fmt.Sprintf("Ditolak oleh Wali Kelas (%s)", teacher.Nama)
	}

	updates := map[string]interface{}{
		"status":          "rejected",
		"rejected_reason": finalReason,
		"approved_at":     now,
		"updated_at":      now,
	}
	if teacher.UserID != nil && *teacher.UserID > 0 {
		updates["approved_by"] = *teacher.UserID
	}

	if err := config.DB.Table("student_leaves").Where("id = ?", leave.ID).Updates(updates).Error; err != nil {
		return fmt.Sprintf("❌ *Gagal Menolak Izin*\n\nTerjadi kesalahan database: %v", err)
	}

	// Kirim notifikasi WhatsApp penolakan ke Orang Tua / Siswa secara background
	go func() {
		targetSchoolID := schoolID
		if targetSchoolID == "" || targetSchoolID == "1" {
			targetSchoolID = fmt.Sprintf("%d", leave.SchoolID)
		}
		notifMsg := GenerateParentLeaveRejectedNotification(
			leave.NamaSiswa,
			leave.Jenis,
			leave.TanggalMulai,
			leave.TanggalSelesai,
			teacher.Nama,
			finalReason,
		)

		phones := getUniquePhones(leave.NoWaPengaju, leave.WaOrtu, leave.NoWaSiswa)
		for _, p := range phones {
			SendMessage(p, notifMsg, targetSchoolID)
		}
	}()

	dateRange := FormatDateRangeIndo(leave.TanggalMulai, leave.TanggalSelesai)
	return fmt.Sprintf("❌ *Pengajuan Izin Ditolak*\n\nPermohonan *%s* untuk siswa:\n👤 Nama: *%s*\n🏫 Kelas: *%s*\n📅 Tanggal: *%s*\n📌 Alasan: %s\n\nNotifikasi penolakan telah diteruskan ke nomor pengaju/orang tua.", strings.Title(leave.Jenis), leave.NamaSiswa, leave.NamaKelas, dateRange, finalReason)
}

func ListPendingLeavesForTeacher(teacher *Guru, schoolID string) string {
	var leaves []StudentLeave
	query := config.DB.Table("student_leaves l").
		Select("l.*, s.nama as nama_siswa, s.nis, s.no_wa as no_wa_siswa, s.wa_ortu, s.kelas_id, k.nama_kelas, k.wali_kelas_id, k.wali_kelas_2_id").
		Joins("INNER JOIN siswa s ON l.student_id = s.id").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("l.status = 'pending'")

	if teacher.SchoolID != 0 {
		query = query.Where("l.school_id = ?", teacher.SchoolID)
	}

	if !teacher.IsGlobalReport {
		query = query.Where("(k.wali_kelas_id = ? OR k.wali_kelas_2_id = ?)", teacher.ID, teacher.ID)
	}

	result := query.Order("l.created_at DESC").Limit(10).Find(&leaves)
	if result.Error != nil || len(leaves) == 0 {
		return fmt.Sprintf("📋 *Daftar Pengajuan Izin Menunggu Persetujuan*\n\nSaat ini *tidak ada* pengajuan izin berstatus pending di kelas yang Anda ampu (%s).\n\nSemua pengajuan telah selesai diproses.", teacher.Nama)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 *Pengajuan Izin Menunggu Persetujuan (%d)*\n", len(leaves)))
	sb.WriteString(fmt.Sprintf("Wali Kelas: *%s*\n\n", teacher.Nama))

	for i, l := range leaves {
		dateRange := FormatDateRangeIndo(l.TanggalMulai, l.TanggalSelesai)
		sb.WriteString(fmt.Sprintf("%d. *%s* (%s)\n", i+1, l.NamaSiswa, l.NamaKelas))
		sb.WriteString(fmt.Sprintf("   • Jenis: *%s*\n", strings.Title(l.Jenis)))
		sb.WriteString(fmt.Sprintf("   • Tanggal: %s\n", dateRange))
		sb.WriteString(fmt.Sprintf("   • Alasan: %s\n", l.Keterangan))
		sb.WriteString(fmt.Sprintf("   • Kode: `%s`\n\n", l.Code))
	}

	sb.WriteString("⚡ *Cara Memproses:*\n")
	sb.WriteString("• Balas: `acc <kode>` untuk menyetujui\n")
	sb.WriteString("• Balas: `tolak <kode> [alasan]` untuk menolak\n")
	sb.WriteString(fmt.Sprintf("\n_Contoh: `acc %s`_", leaves[0].Code))

	return sb.String()
}
