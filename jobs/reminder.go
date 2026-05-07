package jobs

import (
	"fmt"
	"log"
	"time"

	"bot_wa/config"
	"bot_wa/services"
)

// ─── Models ───────────────────────────────────────────────────────────────────

type TeacherReminder struct {
	TeacherID    uint
	TeacherName  string
	TeacherPhone string
	SchoolID     string
	Students     []map[string]string
}

type Schedule struct {
	SchoolID  uint   `gorm:"column:school_id"`
	Hari      string `gorm:"column:hari"`
	IndexHari int    `gorm:"column:index_hari"`
	JamPulang string `gorm:"column:jam_pulang"`
}

// ─── Query ────────────────────────────────────────────────────────────────────

func loadSchedules() []Schedule {
	var schedules []Schedule
	config.DB.Table("jadwal").
		Select("school_id, hari, index_hari, jam_pulang").
		Where("is_active = 1 AND school_id IS NOT NULL").
		Scan(&schedules)
	return schedules
}

func getStudentsNeedingCheckout(schoolID string) ([]TeacherReminder, error) {
	todayStr := services.GetToday()

	type Row struct {
		TeacherID    uint   `gorm:"column:teacher_id"`
		TeacherName  string `gorm:"column:teacher_name"`
		TeacherPhone string `gorm:"column:teacher_phone"`
		SchoolID     uint   `gorm:"column:school_id"`
		StudentName  string `gorm:"column:student_name"`
		ClassName    string `gorm:"column:class_name"`
		JamMasuk     string `gorm:"column:jam_masuk"`
	}

	var rows []Row
	config.DB.Table("attendance a").
		Select("g.id as teacher_id, g.nama as teacher_name, g.no_wa as teacher_phone, g.school_id, s.nama as student_name, k.nama_kelas as class_name, a.jam_masuk").
		Joins("INNER JOIN siswa s ON a.student_id = s.id").
		Joins("INNER JOIN guru g ON a.checked_in_by_teacher_id = g.id").
		Joins("LEFT JOIN kelas k ON s.kelas_id = k.id").
		Where("a.tanggal = ? AND a.jam_masuk IS NOT NULL AND a.jam_pulang IS NULL AND a.checked_in_by_teacher_id IS NOT NULL AND g.school_id = ?", todayStr, schoolID).
		Order("g.id, s.nama").
		Scan(&rows)

	teacherMap := make(map[uint]*TeacherReminder)
	for _, r := range rows {
		t, ok := teacherMap[r.TeacherID]
		if !ok {
			teacherMap[r.TeacherID] = &TeacherReminder{
				TeacherID:    r.TeacherID,
				TeacherName:  r.TeacherName,
				TeacherPhone: r.TeacherPhone,
				SchoolID:     fmt.Sprintf("%d", r.SchoolID),
				Students:     []map[string]string{},
			}
			t = teacherMap[r.TeacherID]
		}
		t.Students = append(t.Students, map[string]string{
			"name":      r.StudentName,
			"class":     r.ClassName,
			"jam_masuk": r.JamMasuk,
		})
	}

	var result []TeacherReminder
	for _, t := range teacherMap {
		result = append(result, *t)
	}
	return result, nil
}

func sendCheckoutRemindersForSchool(schoolID string) {
	log.Printf("🔔 Running checkout reminder for school %s...", schoolID)
	teachers, err := getStudentsNeedingCheckout(schoolID)
	if err != nil {
		log.Printf("❌ Reminder job failed for school %s: %v", schoolID, err)
		return
	}
	if len(teachers) == 0 {
		log.Printf("✅ No pending checkouts for school %s.", schoolID)
		return
	}

	for _, t := range teachers {
		if t.TeacherPhone == "" {
			continue
		}
		msg := services.GenerateCheckoutReminderMessage(t.TeacherName, t.Students)
		services.SendMessage(t.TeacherPhone, msg, t.SchoolID)
		log.Printf("✅ Reminder sent to %s (%d students, school %s)", t.TeacherName, len(t.Students), t.SchoolID)
		time.Sleep(500 * time.Millisecond)
	}
}

// ─── Scheduler ────────────────────────────────────────────────────────────────

// StartReminderScheduler menjalankan scheduler yang reload jadwal setiap hari.
// Tiap sekolah punya jadwal pulang sendiri — reminder dikirim per sekolah
// sesuai jam pulang masing-masing.
func StartReminderScheduler() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	log.Println("📅 Checkout reminder scheduler started")

	go func() {
		for {
			schedules := loadSchedules()
			if len(schedules) == 0 {
				log.Println("⚠️ No active schedule found, retrying in 1 hour...")
				time.Sleep(1 * time.Hour)
				continue
			}
			log.Printf("📋 Loaded %d schedule(s) for %d school(s)", len(schedules), countSchools(schedules))

			// Jalankan satu hari, lalu return → reload jadwal untuk hari berikutnya
			runDailySchedules(schedules, loc)
		}
	}()
}

// runDailySchedules memonitor waktu selama satu hari.
// Saat jam pulang suatu sekolah tiba, reminder dikirim hanya ke guru sekolah tersebut.
func runDailySchedules(schedules []Schedule, loc *time.Location) {
	now := time.Now().In(loc)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, loc)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// fired mencegah double-trigger jadwal yang sama dalam satu hari
	// key: "schoolID-scheduleIndex"
	fired := make(map[string]bool)

	for {
		t := <-ticker.C
		current := t.In(loc)

		if current.After(endOfDay) {
			log.Println("📅 New day detected, reloading schedules from database...")
			return
		}

		for i, s := range schedules {
			key := fmt.Sprintf("%d-%d", s.SchoolID, i)
			if fired[key] {
				continue
			}

			var h, m int
			if len(s.JamPulang) >= 5 {
				parseInt2(&h, s.JamPulang[0:2])
				parseInt2(&m, s.JamPulang[3:5])
			}

			// node-cron: 1=Mon…7=Sun → Go: 0=Sun…6=Sat
			targetWeekday := s.IndexHari % 7
			if int(current.Weekday()) == targetWeekday &&
				current.Hour() == h &&
				current.Minute() == m {
				fired[key] = true
				schoolID := fmt.Sprintf("%d", s.SchoolID)
				log.Printf("⏰ Firing reminder: school %s — %s at %02d:%02d", schoolID, s.Hari, h, m)
				go sendCheckoutRemindersForSchool(schoolID)
			}
		}
	}
}

func countSchools(schedules []Schedule) int {
	seen := make(map[uint]bool)
	for _, s := range schedules {
		seen[s.SchoolID] = true
	}
	return len(seen)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseInt2(dst *int, s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseErr{}
		}
		n = n*10 + int(c-'0')
	}
	*dst = n
	return n, nil
}

type parseErr struct{}

func (e *parseErr) Error() string { return "parse error" }
