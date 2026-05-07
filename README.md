# WhatsApp Bot Absensi (Go Version)

Bot WhatsApp ini merupakan layanan interaktif yang dibangun menggunakan **Go (Golang)** dan **Fiber Framework**. Bot ini ditugaskan untuk menangani webhook dari WhatsApp (GoWA) dan berinteraksi secara otomatis dengan guru-guru dalam mengelola absensi siswa, memberikan notifikasi, serta mengatur pengingat otomatis.

Bot ini dirancang dengan kapabilitas **Multi-Tenant** (Mendukung banyak sekolah / device secara simultan) dan menggunakan mekanisme **Session-Aware** untuk percakapan bot yang bertahap (State Machine).

---

## 🚀 Fitur Utama

1. **Multi-Tenant (Banyak Sekolah & Device)**
   - Mendeteksi ID Perangkat (`X-Device-Id`) atau `DeviceID` dari payload webhook untuk memetakan sekolah.
   - Device mapping divalidasi dan di-cache (termasuk konversi JID ke ID perangkat secara dinamis) dengan metode fallback untuk stabilitas operasional.

2. **Session-Aware (Percakapan Bertahap)**
   - Mendukung sesi berjangka waktu (berakhir dalam 2 menit jika tanpa respons).
   - Pembersihan otomatis (Cleanup) sesi kedaluwarsa dengan mengirimkan pesan peringatan kepada pengguna, menghindari kebingungan dalam _state_ pengisian absensi.
   - Sesi tidak berakhir otomatis ketika ada input yang tidak dikenali, memudahkan guru memperbaiki _typo_.

3. **Privacy-Focused (Anti Spam Group)**
   - Otomatis **mengabaikan semua pesan dari grup WhatsApp** dan hanya menanggapi percakapan *Private* langsung dari nomor guru yang terdaftar.

4. **Pengelolaan Absensi Interaktif (Teacher Command)**
   Guru dapat menggunakan beberapa menu interaktif:
   - `search_student`: Pencarian siswa untuk pencatatan absensi.
   - `quick_checkin`: Fitur pencatatan jam masuk cepat.
   - `quick_checkout`: Fitur pencatatan jam pulang cepat.
   - `edit_attendance`: Edit kehadiran atau keterangan.
   - `delete_attendance`: Menghapus riwayat kehadiran yang salah.
   - `recap_student`: Menampilkan rekap kehadiran siswa secara instan.
   - `search_contact`: Mencari informasi kontak wali murid.

5. **Pengingat Jam Pulang Otomatis (Dynamic Scheduler)**
   - Scheduler berjalan terus dan mendeteksi jadwal setiap sekolah secara dinamis berdasarkan database.
   - Mengirim notifikasi *Checkout Reminder* kepada guru untuk siswa yang belum terekam absen pulangnya.
   - Jadwal berjalan sesuai *timezone* lokal.

---

## 📂 Struktur Direktori Utama

- `config/` : Inisialisasi Database (MySQL) dan Konfigurasi Klien HTTP GoWA.
- `handlers/` : Controller/Handler webhook HTTP. Di sinilah payload webhook difilter, parsing, dan sesi divalidasi (`webhook.go`).
- `services/` : Core business logic.
  - `session.go`: Manajamen *in-memory session state*, konversi ID Perangkat, dan pengiriman pesan via WA API.
  - `message.go`: Pembuatan template balasan, format data list/menu interaktif.
  - `attendance.go`: Logika mutasi dan pengambilan data absensi ke/dari database.
- `jobs/` : Goroutines untuk _background process_.
  - `reminder.go`: Scheduler harian untuk _Checkout Reminder_ guru sesuai sekolah masing-masing.
- `main.go` : *Entry point* aplikasi. Inisialisasi Fiber app, middleware, koneksi database, cron jobs, dan *graceful shutdown*.

---

## ⚙️ Kebutuhan Sistem & Dependensi (Requirements)

- **Go** (minimal versi 1.20+)
- **Database:** MySQL / MariaDB (Terhubung langsung ke skema Laravel)
- **Library Utama:**
  - `gofiber/fiber/v2`: Web Framework yang cepat.
  - `joho/godotenv`: Membaca variabel dari `.env`.
  - `gorm.io/gorm`: ORM.

## 🛠 Instalasi dan Menjalankan

1. **Konfigurasi Environment**
   Ganti atau sesuaikan konfigurasi pada file `.env` di root repository.
   ```env
   PORT=3001
   DB_USER=root
   DB_PASS=
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_NAME=absen_multi

   # Konfigurasi Endpoint WA
   WA_API_URL=http://localhost:3000
   WA_API_USER=admin
   WA_API_PASSWORD=admin
   ```

2. **Menjalankan Service**
   ```bash
   # Download semua module
   go mod tidy

   # Menjalankan (Development)
   go run main.go

   # Melakukan Build (Production)
   go build -o bot_wa.exe main.go
   ```

## 🔌 Endpoint API

- `GET /` - Status sederhana.
- `POST /webhook` - Menerima Event Webhook WhatsApp dari aplikasi GoWA (atau provider setara). Membaca payload baik dari root `event` atau sub-payload `payload`.
- `GET /webhook` - Pemeriksaan ping/health status Webhook.

---

## 🔧 Mekanisme Background Job (Scheduler)

Scheduler jam pulang menggunakan arsitektur Goroutine pada Go. Berjalan di file `jobs/reminder.go`.
- Fungsi `StartReminderScheduler()` mengambil jadwal hari ini secara keseluruhan (Semua Sekolah).
- `runDailySchedules()` menahan siklus berjalan (`ticker` 30 detik) dan mencocokkan jam.
- Jika jadwal (jam & menit) cocok dengan jam server dan identitas hari-nya (*index_hari*), maka `sendCheckoutRemindersForSchool()` dipanggil.
- Semua notifikasi masuk ke antrean *async* melalui Goroutine sehingga tidak mem-blokir webhook.

---
_Dokumentasi ini otomatis di-generate berdasarkan versi source code terbaru._
