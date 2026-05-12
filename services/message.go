package services

import (
	"fmt"
	"strings"
	"time"
)

// Jakarta timezone helper
func jakartaNow() time.Time {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc)
}

func formatTimeStr(timeStr string) string {
	if timeStr == "" {
		return "-"
	}
	// timeStr can be "15:04:05" or already "15:04"
	parts := strings.Split(timeStr, ":")
	if len(parts) >= 2 {
		return parts[0] + ":" + parts[1]
	}
	return timeStr
}

func formatDate(t time.Time) string {
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%s, %02d %s %d", days[t.Weekday()], t.Day(), months[t.Month()], t.Year())
}

func formatMonth(t time.Time) string {
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%s %d", months[t.Month()], t.Year())
}

func formatStatusEmoji(status string) string {
	m := map[string]string{
		"H": "✅ Hadir",
		"I": "📝 Izin",
		"S": "🤒 Sakit",
		"A": "❌ Alpha",
		"B": "🚫 Bolos",
		"P": "🏠 Pulang",
	}
	if v, ok := m[status]; ok {
		return v
	}
	return status
}

func formatStatusText(status string) string {
	m := map[string]string{"H": "Hadir", "I": "Izin", "S": "Sakit", "A": "Alpha", "B": "Bolos"}
	if v, ok := m[status]; ok {
		return v
	}
	return status
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "-"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%d jam %d menit", h, m)
	}
	return fmt.Sprintf("%d menit", m)
}

// ─── Command Parsing ──────────────────────────────────────────────────────────

type ParsedCommand struct {
	Command    string
	SearchTerm string
	Period     string
	Option     int
}

func ParseTeacherCommand(msg string) ParsedCommand {
	text := strings.ToLower(strings.TrimSpace(msg))

	if text == "help" || text == "menu" || text == "bantuan" {
		return ParsedCommand{Command: "teacher_help"}
	}
	if strings.HasPrefix(text, "masuk ") {
		return ParsedCommand{Command: "quick_checkin", SearchTerm: strings.TrimPrefix(text, "masuk ")}
	}
	if strings.HasPrefix(text, "pulang ") {
		return ParsedCommand{Command: "quick_checkout", SearchTerm: strings.TrimPrefix(text, "pulang ")}
	}
	if strings.HasPrefix(text, "edit absen ") {
		return ParsedCommand{Command: "edit_attendance", SearchTerm: strings.TrimPrefix(text, "edit absen ")}
	}
	if strings.HasPrefix(text, "ubah absen ") {
		return ParsedCommand{Command: "edit_attendance", SearchTerm: strings.TrimPrefix(text, "ubah absen ")}
	}
	if strings.HasPrefix(text, "hapus absen ") {
		return ParsedCommand{Command: "delete_attendance", SearchTerm: strings.TrimPrefix(text, "hapus absen ")}
	}
	if strings.HasPrefix(text, "delete absen ") {
		return ParsedCommand{Command: "delete_attendance", SearchTerm: strings.TrimPrefix(text, "delete absen ")}
	}
	if strings.HasPrefix(text, "cari ") {
		return ParsedCommand{Command: "search_contact", SearchTerm: strings.TrimPrefix(text, "cari ")}
	}
	if strings.HasPrefix(text, "kontak ") {
		return ParsedCommand{Command: "search_contact", SearchTerm: strings.TrimPrefix(text, "kontak ")}
	}
	if strings.HasPrefix(text, "info ") {
		return ParsedCommand{Command: "search_contact", SearchTerm: strings.TrimPrefix(text, "info ")}
	}
	if strings.HasPrefix(text, "absen ") {
		return ParsedCommand{Command: "search_student", SearchTerm: strings.TrimPrefix(text, "absen ")}
	}
	if strings.HasPrefix(text, "rekap ") {
		return ParsedCommand{Command: "recap_student", SearchTerm: strings.TrimPrefix(text, "rekap ")}
	}
	if text == "ya" || text == "yes" || text == "iya" || text == "oke" || text == "ok" {
		return ParsedCommand{Command: "confirm_yes"}
	}
	if text == "tidak" || text == "no" || text == "batal" || text == "cancel" || text == "nggak" || text == "enggak" {
		return ParsedCommand{Command: "confirm_no"}
	}
	// Numeric
	var n int
	if _, err := fmt.Sscanf(text, "%d", &n); err == nil && fmt.Sprintf("%d", n) == text {
		return ParsedCommand{Command: "select_option", Option: n}
	}

	return ParsedCommand{Command: "unknown"}
}

// ParsePublicCommand parses perintah dari publik (siswa/ortu yang belum terdaftar)
func ParsePublicCommand(msg string) ParsedCommand {
	text := strings.ToLower(strings.TrimSpace(msg))

	if text == "daftar siswa" {
		return ParsedCommand{Command: "register_student"}
	}
	if text == "daftar ortu" || text == "daftar orang tua" || text == "daftar wali" {
		return ParsedCommand{Command: "register_parent"}
	}
	if text == "ya" || text == "yes" || text == "iya" || text == "oke" || text == "ok" {
		return ParsedCommand{Command: "confirm_yes"}
	}
	if text == "tidak" || text == "no" || text == "batal" || text == "cancel" || text == "nggak" || text == "enggak" {
		return ParsedCommand{Command: "confirm_no"}
	}
	var n int
	if _, err := fmt.Sscanf(text, "%d", &n); err == nil && fmt.Sprintf("%d", n) == text {
		return ParsedCommand{Command: "select_option", Option: n}
	}
	return ParsedCommand{Command: "unknown"}
}

// ─── Message Generators ───────────────────────────────────────────────────────

func GenerateTeacherHelpMessage(teacherName, schoolName string) string {
	return fmt.Sprintf(`👋 *Halo, %s!*

Selamat datang di *Bot Absensi %s* 🎓

📋 *Daftar Perintah:*

1️⃣ *Absen Manual Siswa*
   Ketik: `+"`"+`absen [nama siswa]`+"`"+`
   Contoh: `+"`"+`absen andi`+"`"+`
   
2️⃣ *Catat Jam Masuk*
   Ketik: `+"`"+`masuk [nama siswa]`+"`"+`
   Contoh: `+"`"+`masuk andi`+"`"+`
   
3️⃣ *Catat Jam Pulang*
   Ketik: `+"`"+`pulang [nama siswa]`+"`"+`
   Contoh: `+"`"+`pulang andi`+"`"+`
   
4️⃣ *Cari Kontak Siswa/Ortu*
   Ketik: `+"`"+`cari [nama]`+"`"+` atau `+"`"+`kontak [nama]`+"`"+`
   Contoh: `+"`"+`cari andi`+"`"+`

5️⃣ *Edit Absensi*
   Ketik: `+"`"+`edit absen [nama siswa]`+"`"+`
   
6️⃣ *Hapus Absensi*
   Ketik: `+"`"+`hapus absen [nama siswa]`+"`"+`

7️⃣ *Rekap Siswa (Bulan Ini)*
   Ketik: `+"`"+`rekap [nama siswa]`+"`"+`
   
8️⃣ *Menu Bantuan*
   Ketik: `+"`"+`help`+"`"+` atau `+"`"+`menu`+"`"+`

💡 _Cukup ketik perintah di atas untuk menggunakan bot ini._

📚 Semangat mengajar! ✨`, teacherName, schoolName)
}

func GenerateStudentSearchResults(students []Siswa, searchTerm string) string {
	if len(students) == 0 {
		return fmt.Sprintf(`❌ *Siswa Tidak Ditemukan*

Tidak ada siswa dengan nama yang mengandung "*%s*".

Silakan coba dengan nama lain atau ketik `+"`"+`help`+"`"+` untuk bantuan.`, searchTerm)
	}

	msg := fmt.Sprintf("🔍 *Hasil Pencarian: \"%s\"*\n\nDitemukan %d siswa:\n\n", searchTerm, len(students))
	for i, s := range students {
		kelas := s.NamaKelas
		if kelas == "" {
			kelas = "-"
		}
		msg += fmt.Sprintf("%d. *%s*\n   📚 Kelas: %s\n   🆔 NISN: %s\n\n", i+1, s.Nama, kelas, s.NIS)
	}
	msg += fmt.Sprintf("\n💡 _Balas dengan nomor siswa yang ingin diabsen (1-%d)_", len(students))
	return msg
}

func GenerateStatusSelectionMessage(studentName string) string {
	return fmt.Sprintf(`📝 *Pilih Status Absensi*

Siswa: *%s*

Pilih status absensi:

1️⃣ Hadir
2️⃣ Izin
3️⃣ Sakit
4️⃣ Alpha

💡 _Balas dengan nomor status (1-4)_`, studentName)
}

func GenerateKeteranganInputMessage(studentName, status string) string {
	statusText := map[string]string{"H": "Hadir", "I": "Izin", "S": "Sakit", "A": "Alpha"}
	return fmt.Sprintf(`📝 *Masukkan Keterangan*

Siswa: *%s*
Status: *%s*

Silakan ketik keterangan untuk absensi ini.

Contoh:
- "Sakit demam"
- "Izin keperluan keluarga"

💡 _Ketik keterangan_`, studentName, statusText[status])
}

func GenerateAttendanceConfirmation(studentName, studentClass, status, keterangan string) string {
	emoji := map[string]string{"H": "✅", "I": "📝", "S": "🤒", "A": "❌"}
	now := jakartaNow()
	return fmt.Sprintf(`%s *Absensi Berhasil Dicatat*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam: %s
📊 Status: *%s*
📝 Keterangan: %s

_Absensi telah tersimpan dalam sistem._ ✨

Ketik `+"`"+`absen [nama]`+"`"+` untuk absen siswa lain.`,
		emoji[status], studentName, orDash(studentClass),
		formatDate(now), now.Format("15:04"),
		formatStatusText(status), keterangan)
}

func GenerateEditOptionsMessage(studentName string) string {
	return fmt.Sprintf(`✏️ *Edit Absensi*

Siswa: *%s*

Apa yang ingin diubah?

1️⃣ Status (Izin/Sakit/Alpha)
2️⃣ Keterangan

💡 _Balas dengan nomor pilihan (1-2)_`, studentName)
}

func GenerateDeleteConfirmationMessage(student *Siswa, att *Attendance) string {
	return fmt.Sprintf(`⚠️ *Konfirmasi Hapus Absensi*

👤 Nama: *%s*
📚 Kelas: %s
📊 Status: %s
📝 Keterangan: %s

Apakah Anda yakin ingin menghapus absensi ini?

💡 _Ketik "ya" untuk hapus atau "tidak" untuk batal_`,
		student.Nama, orDash(student.NamaKelas),
		formatStatusText(att.Status), orDash(att.Keterangan))
}

func GenerateAttendanceExistsConfirmation(student *Siswa, att *Attendance) string {
	now := jakartaNow()
	return fmt.Sprintf(`⚠️ *Absensi Sudah Ada*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s

📋 *Data Absensi Saat Ini:*
⏰ Jam Masuk: %s
📊 Status: *%s*
📝 Keterangan: %s

━━━━━━━━━━━━━━━━━━━━
Apakah Anda ingin *mengganti* data absensi ini?

💡 _Ketik "ya" untuk mengganti atau "tidak" untuk batal_`,
		student.Nama, orDash(student.NamaKelas), formatDate(now),
		formatTimeStr(att.JamMasuk), formatStatusText(att.Status), orDash(att.Keterangan))
}

func GenerateCheckinExistsConfirmation(student *Siswa, att *Attendance) string {
	now := jakartaNow()
	return fmt.Sprintf(`⚠️ *Siswa Sudah Absen Masuk*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s

📋 *Data Absensi Saat Ini:*
⏰ Jam Masuk: %s
📊 Status: *%s*
📝 Keterangan: %s

━━━━━━━━━━━━━━━━━━━━
Apakah Anda ingin *mengganti* jam masuk?

💡 _Ketik "ya" untuk mengganti atau "tidak" untuk batal_`,
		student.Nama, orDash(student.NamaKelas), formatDate(now),
		formatTimeStr(att.JamMasuk), formatStatusText(att.Status), orDash(att.Keterangan))
}

func GenerateCheckoutExistsConfirmation(student *Siswa, att *Attendance) string {
	now := jakartaNow()
	return fmt.Sprintf(`⚠️ *Siswa Sudah Absen Pulang*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s

📋 *Data Absensi Saat Ini:*
⏰ Jam Masuk: %s
🏠 Jam Pulang: %s

━━━━━━━━━━━━━━━━━━━━
Apakah Anda ingin *mengganti* jam pulang?

💡 _Ketik "ya" untuk mengganti atau "tidak" untuk batal_`,
		student.Nama, orDash(student.NamaKelas), formatDate(now),
		formatTimeStr(att.JamMasuk), formatTimeStr(att.JamPulang))
}

func GenerateNoAttendanceFoundMessage(searchTerm string) string {
	return fmt.Sprintf(`❌ *Tidak Ada Absensi*

Tidak ditemukan siswa dengan nama "*%s*" yang memiliki absensi hari ini.

Pastikan siswa sudah diabsen terlebih dahulu.`, searchTerm)
}

func GenerateNoAttendanceForCheckout(searchTerm string) string {
	return fmt.Sprintf("❌ *Belum Ada Absen Masuk*\n\nTidak ditemukan siswa dengan nama \"*%s*\" yang sudah absen masuk hari ini.\n\nSilakan absen masuk terlebih dahulu dengan:\n`masuk [nama]` atau `absen [nama]`", searchTerm)
}

func GenerateQuickCheckinSuccess(studentName, studentClass string) string {
	now := jakartaNow()
	return fmt.Sprintf(`✅ *Absen Masuk Berhasil*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam Masuk: %s
📊 Status: *Hadir*

_Absensi telah tersimpan dalam sistem._ ✨`,
		studentName, orDash(studentClass), formatDate(now), now.Format("15:04"))
}

func GenerateQuickCheckoutSuccess(studentName, studentClass, jamMasuk string) string {
	now := jakartaNow()
	return fmt.Sprintf(`✅ *Absen Pulang Berhasil*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam Masuk: %s
🏠 Jam Pulang: %s

_Absensi telah tersimpan dalam sistem._ ✨`,
		studentName, orDash(studentClass), formatDate(now),
		formatTimeStr(jamMasuk), now.Format("15:04"))
}

func GenerateInvalidSelectionMessage(maxOption int) string {
	return fmt.Sprintf("❌ *Pilihan Tidak Valid*\n\nSilakan balas dengan nomor yang valid (1-%d).\n\nAtau ketik `help` untuk bantuan.", maxOption)
}

// GenerateSessionAlertMessage menghasilkan pesan peringatan kontekstual
// berdasarkan step sesi yang sedang aktif. Dipanggil saat input tidak dikenali
// selama sesi berlangsung, agar sesi tidak dihapus dan user bisa melanjutkan.
func GenerateSessionAlertMessage(session *Session) string {
	switch session.Step {
	case "select_student":
		return "⚠️ *Pilihan Tidak Valid*\n\nSilakan balas dengan *angka* dari daftar siswa di atas.\n\nContoh: ketik *1* untuk siswa pertama.\n\nKetik `help` untuk membatalkan dan keluar dari sesi."
	case "select_status", "edit_status":
		return "⚠️ *Pilihan Tidak Valid*\n\nSilakan balas dengan angka status berikut:\n\n1️⃣ Hadir\n2️⃣ Izin\n3️⃣ Sakit\n4️⃣ Alpha\n\nKetik `help` untuk membatalkan."
	case "select_edit_type":
		return "⚠️ *Pilihan Tidak Valid*\n\nSilakan balas dengan:\n\n1️⃣ Edit Status\n2️⃣ Edit Keterangan\n\nKetik `help` untuk membatalkan."
	case "input_keterangan", "edit_keterangan":
		return "⚠️ *Keterangan tidak boleh kosong*\n\nSilakan ketik keterangan untuk absensi ini.\n\nKetik `help` untuk membatalkan."
	case "confirm_delete", "confirm_replace_create", "confirm_replace_checkin", "confirm_replace_checkout":
		return "⚠️ *Jawaban Tidak Dikenali*\n\nSilakan ketik *ya* untuk konfirmasi atau *tidak* untuk membatalkan."
	default:
		return "⚠️ *Input Tidak Dikenali*\n\nSesi Anda masih aktif, silakan lanjutkan sesuai instruksi sebelumnya.\n\nKetik `help` untuk membatalkan dan kembali ke menu utama."
	}
}

func GenerateEditSuccessMessage(studentName, editType, newValue string) string {
	label := "Status"
	if editType == "keterangan" {
		label = "Keterangan"
	}
	return fmt.Sprintf(`✅ *Absensi Berhasil Diubah*

👤 Nama: *%s*
📝 %s diubah menjadi: *%s*

_Perubahan telah tersimpan dalam sistem._ ✨

Ketik `+"`"+`help`+"`"+` untuk melihat perintah lainnya.`, studentName, label, newValue)
}

func GenerateDeleteSuccessMessage(studentName string) string {
	return fmt.Sprintf("✅ *Absensi Berhasil Dihapus*\n\n👤 Nama: *%s*\n\n_Absensi telah dihapus dari sistem._ ✨\n\nKetik `help` untuk melihat perintah lainnya.", studentName)
}

func GenerateContactInfo(students []Siswa, searchTerm string) string {
	if len(students) == 0 {
		return fmt.Sprintf("❌ Siswa dengan nama \"%s\" tidak ditemukan.", searchTerm)
	}
	msg := fmt.Sprintf("🔍 *HASIL PENCARIAN KONTAK*\nKata kunci: \"%s\"\n\n", searchTerm)
	for i, s := range students {
		noWa := s.NoWa
		if noWa == "" {
			noWa = "-"
		}
		waOrtu := s.WaOrtu
		if waOrtu == "" {
			waOrtu = "-"
		}
		msg += fmt.Sprintf("%d. *%s*\n   🆔 NISN: %s\n   🏫 Kelas: %s\n   📱 Siswa: %s\n   👨‍👩‍👧 Ortu: %s\n", i+1, s.Nama, s.NIS, orDash(s.NamaKelas), noWa, waOrtu)
		if s.NoWa != "" {
			msg += fmt.Sprintf("   💬 Chat Siswa: https://wa.me/%s\n", s.NoWa)
		}
		if s.WaOrtu != "" {
			msg += fmt.Sprintf("   💬 Chat Ortu: https://wa.me/%s\n", s.WaOrtu)
		}
		msg += "\n"
	}
	return msg
}

func GenerateErrorMessage() string {
	return "⚠️ *Terjadi Kesalahan*\n\nMaaf, terjadi kesalahan saat memproses permintaan Anda.\n\nSilakan coba lagi dalam beberapa saat atau hubungi admin jika masalah berlanjut."
}

func GenerateRecapMessage(student *Siswa, stats *AttendanceStats, period string) string {
	now := jakartaNow()
	var periodText, dateRange string
	switch period {
	case "week":
		periodText = "Minggu Ini"
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -(weekday - 1))
		end := now.AddDate(0, 0, 7-weekday)
		months := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
		dateRange = fmt.Sprintf("%02d %s - %02d %s %d", start.Day(), months[start.Month()], end.Day(), months[end.Month()], end.Year())
	case "month":
		periodText = "Bulan Ini"
		dateRange = formatMonth(now)
	default:
		periodText = "Hari Ini"
		dateRange = formatDate(now)
	}

	msg := fmt.Sprintf(`📊 *Rekap Absensi %s*

👤 Nama: *%s*
🎓 Kelas: %s
📅 Periode: %s

📈 *Statistik:*
✅ Hadir: %d
📝 Izin: %d
🤒 Sakit: %d
❌ Alpha: %d
🚫 Bolos: %d
━━━━━━━━━━━━━━
📊 Total: %d hari`,
		periodText, student.Nama, orDash(student.NamaKelas), dateRange,
		stats.Hadir, stats.Izin, stats.Sakit, stats.Alpha, stats.Bolos, stats.Total)

	if len(stats.Records) > 0 && len(stats.Records) <= 7 {
		msg += "\n\n📋 *Detail Absensi:*\n"
		for i, r := range stats.Records {
			msg += fmt.Sprintf("\n%d. %s\n   %s\n   🕐 %s - %s",
				i+1, r.Tanggal,
				formatStatusEmoji(r.Status),
				formatTimeStr(r.JamMasuk), formatTimeStr(r.JamPulang))
		}
	}
	msg += "\n\n_Tetap semangat belajar!_ 💪📚"
	return msg
}

// ─── Notification Messages ────────────────────────────────────────────────────

func GenerateStudentAttendanceNotification(name, status, keterangan, teacherName string) string {
	emoji := map[string]string{"H": "✅", "I": "📝", "S": "🤒", "A": "❌"}
	now := jakartaNow()
	return fmt.Sprintf(`%s *Notifikasi Absensi*

👋 Halo, *%s*!

Guru telah mencatat absensi Anda:

📅 Tanggal: %s
⏰ Jam: %s
📊 Status: *%s*
📝 Keterangan: %s
👨‍🏫 Dicatat oleh: %s

_Pastikan data absensi Anda sudah benar._`,
		emoji[status], name, formatDate(now), now.Format("15:04"),
		formatStatusText(status), keterangan, teacherName)
}

func GenerateStudentCheckinNotification(name, teacherName string) string {
	now := jakartaNow()
	return fmt.Sprintf(`✅ *Notifikasi Absen Masuk*

👋 Halo, *%s*!

Guru telah mencatat absen masuk Anda:

📅 Tanggal: %s
⏰ Jam Masuk: %s
📊 Status: *Hadir*
👨‍🏫 Dicatat oleh: %s

_Jangan lupa absen pulang ya!_ 🏠`, name, formatDate(now), now.Format("15:04"), teacherName)
}

func GenerateStudentCheckoutNotification(name, jamMasuk, teacherName string) string {
	now := jakartaNow()
	return fmt.Sprintf(`🏠 *Notifikasi Absen Pulang*

👋 Halo, *%s*!

Guru telah mencatat absen pulang Anda:

📅 Tanggal: %s
⏰ Jam Masuk: %s
🏠 Jam Pulang: %s
👨‍🏫 Dicatat oleh: %s

_Hati-hati di jalan, sampai jumpa besok!_ 👋`, name, formatDate(now), formatTimeStr(jamMasuk), now.Format("15:04"), teacherName)
}

func GenerateParentAttendanceNotification(name, class, status, keterangan, teacherName string) string {
	emoji := map[string]string{"H": "✅", "I": "📝", "S": "🤒", "A": "❌", "B": "🚫"}
	now := jakartaNow()
	return fmt.Sprintf(`%s *Notifikasi Absensi Siswa*

🏫 Yth. Wali Murid *%s*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam: %s
📊 Status: *%s*
📝 Keterangan: %s
👨‍🏫 Dicatat oleh: %s

_Terima kasih atas perhatiannya._`,
		emoji[status], name, name, orDash(class),
		formatDate(now), now.Format("15:04"),
		formatStatusText(status), keterangan, teacherName)
}

func GenerateParentCheckinNotification(name, class, teacherName string) string {
	now := jakartaNow()
	return fmt.Sprintf(`✅ *Notifikasi Absen Masuk*

🏫 Yth. Wali Murid *%s*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam Masuk: %s
📊 Status: *Hadir*
👨‍🏫 Dicatat oleh: %s

_Terima kasih atas perhatiannya._`,
		name, name, orDash(class), formatDate(now), now.Format("15:04"), teacherName)
}

func GenerateParentCheckoutNotification(name, class, jamMasuk, teacherName string) string {
	now := jakartaNow()
	return fmt.Sprintf(`🏠 *Notifikasi Absen Pulang*

🏫 Yth. Wali Murid *%s*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
⏰ Jam Masuk: %s
🏠 Jam Pulang: %s
👨‍🏫 Dicatat oleh: %s

_Hati-hati di jalan. Terima kasih._`,
		name, name, orDash(class), formatDate(now), formatTimeStr(jamMasuk), now.Format("15:04"), teacherName)
}

func GenerateParentEditNotification(name, class, status, keterangan, teacherName string) string {
	now := jakartaNow()
	return fmt.Sprintf(`✏️ *Notifikasi Perubahan Absensi*

🏫 Yth. Wali Murid *%s*

👤 Nama: *%s*
📚 Kelas: %s
📅 Tanggal: %s
📊 Status Baru: *%s*
📝 Keterangan: %s
👨‍🏫 Diubah oleh: %s

_Terima kasih atas perhatiannya._`,
		name, name, orDash(class), formatDate(now),
		formatStatusText(status), orDash(keterangan), teacherName)
}

func GenerateCheckoutReminderMessage(teacherName string, students []map[string]string) string {
	now := jakartaNow()
	_ = now
	msg := fmt.Sprintf("🔔 *Pengingat Absen Pulang*\n\nHalo, %s!\n\nBerikut siswa yang sudah Anda absen masuk tapi belum absen pulang:\n\n", teacherName)
	for i, s := range students {
		msg += fmt.Sprintf("%d. *%s*\n   📚 Kelas: %s\n   ⏰ Masuk: %s\n\n", i+1, s["name"], orDash(s["class"]), formatTimeStr(s["jam_masuk"]))
	}
	if len(students) > 0 {
		msg += fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━\n💡 Silakan absen pulang dengan:\n`pulang [nama siswa]`\n\nContoh: `pulang %s`\n\nTerima kasih! 🙏", students[0]["name"])
	}
	return msg
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ─── Registration Messages ────────────────────────────────────────────────────

func GeneratePublicWelcomeMessage(schoolName string) string {
	return fmt.Sprintf(`👋 *Selamat Datang di Bot Absensi %s* 🎓

Halo! Nomor Anda belum terdaftar di sistem kami.

Silakan daftar untuk mendapatkan notifikasi absensi:

👨‍🎓 *Untuk Siswa:*
   Ketik: `+"`"+`Daftar Siswa`+"`"+`

👨‍👩‍👧 *Untuk Orang Tua/Wali:*
   Ketik: `+"`"+`Daftar Ortu`+"`"+`

💡 _Setelah terdaftar, Anda akan mendapat notifikasi absensi otomatis._`, schoolName)
}

func GenerateRegisterAskNISMessage(regType string) string {
	if regType == "siswa" {
		return "👨‍🎓 *Pendaftaran Nomor WA Siswa*\n\n🆔 Silakan ketik *NISN* Anda.\n\nContoh: `2024001`\n\n💡 _Ketik NISN Anda sekarang_"
	}
	return "👨‍👩‍👧 *Pendaftaran Nomor WA Orang Tua/Wali*\n\n🆔 Silakan ketik *NISN* anak Anda.\n\nContoh: `2024001`\n\n💡 _Ketik NISN anak Anda sekarang_"
}

func GenerateRegisterAskTglLahirMessage(student *Siswa, regType string) string {
	who := "Anda"
	if regType == "ortu" {
		who = "anak Anda"
	}
	return fmt.Sprintf("✅ *Siswa Ditemukan*\n\n👤 Nama: *%s*\n📚 Kelas: %s\n🆔 NISN: %s\n\n🔐 Untuk verifikasi, masukkan *tanggal lahir %s*.\n\nFormat: `DD-MM-YYYY`\nContoh: `15-08-2007`\n\n💡 _Ketik tanggal lahir sekarang_",
		student.Nama, orDash(student.NamaKelas), student.NIS, who)
}

func GenerateRegisterNISNotFoundMessage(nis, regType string) string {
	cmd := "Daftar Siswa"
	if regType == "ortu" {
		cmd = "Daftar Ortu"
	}
	return fmt.Sprintf("❌ *NISN Tidak Ditemukan*\n\nNISN *%s* tidak terdaftar di sistem kami.\n\nSilakan periksa kembali NISN Anda dan coba lagi.\n\nKetik `%s` untuk memulai ulang.", nis, cmd)
}

func GenerateRegisterTglLahirWrongMessage(regType string) string {
	cmd := "Daftar Siswa"
	if regType == "ortu" {
		cmd = "Daftar Ortu"
	}
	return fmt.Sprintf("❌ *Tanggal Lahir Tidak Sesuai*\n\nTanggal lahir yang Anda masukkan tidak cocok dengan data kami.\n\nSilakan periksa kembali, atau hubungi admin sekolah jika ada kesalahan data.\n\nKetik `%s` untuk mencoba kembali.", cmd)
}

func GenerateRegisterSuccessMessage(studentName, regType, schoolName string) string {
	if regType == "siswa" {
		return fmt.Sprintf(`🎉 *Pendaftaran Berhasil!*

👤 Nama: *%s*
🏫 Sekolah: %s

Nomor WhatsApp Anda telah berhasil didaftarkan sebagai *nomor siswa*.

Sekarang Anda akan mendapat notifikasi otomatis ketika:
✅ Hadir / Jam masuk dicatat
🏠 Jam pulang dicatat
📊 Status absensi diperbarui

_Terima kasih telah mendaftar!_ 🙏`, studentName, schoolName)
	}
	return fmt.Sprintf(`🎉 *Pendaftaran Berhasil!*

👤 Siswa: *%s*
🏫 Sekolah: %s

Nomor WhatsApp Anda telah berhasil didaftarkan sebagai *nomor orang tua/wali*.

Sekarang Anda akan mendapat notifikasi otomatis ketika:
✅ Hadir / Jam masuk dicatat
🏠 Jam pulang dicatat
📊 Status absensi diperbarui

_Terima kasih telah mendaftar!_ 🙏`, studentName, schoolName)
}

func GenerateRegisterCancelMessage() string {
	return "❌ *Pendaftaran Dibatalkan*\n\nPendaftaran nomor WA dibatalkan.\n\nKetik `Daftar Siswa` atau `Daftar Ortu` jika ingin mendaftar kembali."
}

func GenerateRegisterSessionAlert(step string) string {
	switch step {
	case "register_ask_nis":
		return "⚠️ *Input Tidak Valid*\n\nSilakan ketik *NISN* (Nomor Induk Siswa) yang valid.\n\nContoh: `2024001`"
	case "register_ask_tgl_lahir":
		return "⚠️ *Format Tidak Valid*\n\nSilakan masukkan tanggal lahir dengan format:\n`DD-MM-YYYY`\n\nContoh: `15-08-2007`"
	default:
		return "⚠️ *Input Tidak Dikenali*\n\nKetik `Daftar Siswa` atau `Daftar Ortu` untuk memulai pendaftaran."
	}
}
