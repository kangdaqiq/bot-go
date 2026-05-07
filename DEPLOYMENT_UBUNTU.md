# Panduan Deployment di Server Ubuntu

Dokumentasi ini menjelaskan langkah-langkah untuk melakukan *deploy* (menjalankan) WhatsApp Bot Absensi (Go) di server Ubuntu (misal: VPS di Niagahoster, DigitalOcean, AWS, dll), serta menjalankannya di _background_ menggunakan **Systemd**.

---

## 1. Persiapan Awal (Prerequisites)

Pastikan server Anda sudah terinstal beberapa kebutuhan dasar:

```bash
# Update repository server
sudo apt update && sudo apt upgrade -y

# Install Git & Nano
sudo apt install git nano -y
```

> **Catatan:** Anda punya 2 pilihan untuk men-deploy aplikasi Go.
> - **Opsi A**: Anda _build_ file `bot_wa` (binary executable) di laptop Windows/Mac Anda (mengatur target OS ke linux), lalu copy file jadinya ke server (lebih hemat RAM di server).
> - **Opsi B**: Anda menginstal Go di server Ubuntu dan me-compile-nya langsung di server. Panduan ini akan menggunakan **Opsi B**.

---

## 2. Menginstal Golang di Ubuntu

1. Download dan ekstrak Golang versi terbaru (sesuaikan link dengan versi terbaru di situs resmi Go):
   ```bash
   wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
   sudo rm -rf /usr/local/go
   sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
   ```

2. Tambahkan Golang ke Path *environment*:
   ```bash
   echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
   source ~/.profile
   ```

3. Cek apakah instalasi berhasil:
   ```bash
   go version
   ```

---

## 3. Clone Repositori dan Konfigurasi

1. Clone repositori dari GitHub:
   ```bash
   cd ~
   git clone https://github.com/kangdaqiq/bot-go.git
   cd bot-go
   ```

2. Buat file `.env` dengan menduplikat dari `.env.example`:
   ```bash
   cp .env.example .env
   ```

3. Edit file `.env` dengan editor nano:
   ```bash
   nano .env
   ```
   > Sesuaikan kredensial koneksi Database dan URL GoWA API Anda di dalam file ini. Tekan `CTRL + O`, lalu `Enter` untuk menyimpan, dan `CTRL + X` untuk keluar.

---

## 4. Build Aplikasi

Jalankan perintah berikut untuk mengunduh semua *dependencies* dan mem-build aplikasi menjadi file executable (`bot_wa`):

```bash
go mod tidy
go build -o bot_wa main.go
```
Jika berhasil, Anda akan melihat file bernama `bot_wa` (berwarna hijau jika dilihat dengan `ls -l` karena executable).

---

## 5. Membuat Service Systemd (Agar Berjalan di Background)

Agar aplikasi dapat berjalan terus-menerus tanpa harus selalu membuka terminal, dan dapat *auto-restart* jika server mati, kita perlu membuat _service_ di Systemd.

1. Buat file service baru:
   ```bash
   sudo nano /etc/systemd/system/bot_wa.service
   ```

2. Copy dan Paste konfigurasi berikut ke dalam editor nano (ubah `/root/bot-go` dengan lokasi folder clone Anda jika berbeda):

   ```ini
   [Unit]
   Description=WhatsApp Bot Absensi Go Service
   After=network.target

   [Service]
   Type=simple
   # Ubah root dengan username Anda jika tidak menggunakan akun root
   User=root
   WorkingDirectory=/root/bot-go
   ExecStart=/root/bot-go/bot_wa
   Restart=on-failure
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```
   > Simpan file tersebut (`CTRL+O`, `Enter`, `CTRL+X`).

3. Reload Systemd untuk membaca file service yang baru kita buat:
   ```bash
   sudo systemctl daemon-reload
   ```

4. Aktifkan service agar otomatis berjalan setiap kali server (Ubuntu) menyala/reboot:
   ```bash
   sudo systemctl enable bot_wa
   ```

5. Jalankan aplikasi bot:
   ```bash
   sudo systemctl start bot_wa
   ```

---

## 6. Cek Status & Log Aplikasi

Untuk melihat apakah aplikasi sudah berjalan dengan normal:
```bash
sudo systemctl status bot_wa
```

Untuk melihat **Log secara langsung (Real-time)** (misal ketika ingin melihat apakah webhook masuk atau pesan terkirim):
```bash
sudo journalctl -u bot_wa.service -f
```
*(Tekan `CTRL + C` untuk keluar dari tampilan log).*

---

## 7. Melakukan Update Kode (Pull)

Jika di kemudian hari ada pembaruan kode pada GitHub yang ingin diterapkan ke server, cukup lakukan langkah berikut:

```bash
cd ~/bot-go
# Tarik update terbaru
git pull origin main

# Build ulang aplikasinya
go build -o bot_wa main.go

# Restart service agar sistem menggunakan aplikasi yang baru di-build
sudo systemctl restart bot_wa
```

---
**Selesai!** Kini Bot WhatsApp Absensi Anda sudah ter-deploy dan berjalan secara optimal di background server Ubuntu Anda.
