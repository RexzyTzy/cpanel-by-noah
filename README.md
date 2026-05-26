# RenzyDev Panel Manager

Website management panel Pterodactyl + WhatsApp notifikasi via Fonnte.

## 📁 Struktur File

```
pterodactyl-panel/
├── main.go      ← Semua kode ada di sini (1 file)
├── go.mod       ← Go module
└── README.md    ← Panduan ini
```

## ⚙️ Setup Config

Buka `main.go` dan edit bagian CONFIG di baris atas:

```go
const (
    PterodactylURL    = "https://reshhus.myserverr.web.id"       // ✅ sudah diset
    PterodactylAPIKey = "MASUKKAN_API_KEY_ADMIN_PTERODACTYL_DISINI" // ← GANTI INI
    FonnteAPIKey      = "MASUKKAN_API_KEY_FONNTE_DISINI"            // ← GANTI INI
    PanelLink         = "https://reshhus.myserverr.web.id"       // ✅ sudah diset
    PhpMyAdminLink    = "https://reshhus.myserverr.web.id/pma"   // ✅ sudah diset
    DefaultNestID     = 9                                         // ✅ sudah diset
    DefaultEggID      = 29                                        // ✅ sudah diset
)
```

## 🚀 Deploy ke Railway

1. Push ke GitHub:
   ```bash
   git init
   git add .
   git commit -m "init"
   git remote add origin https://github.com/username/repo.git
   git push -u origin main
   ```

2. Di Railway:
   - New Project → Deploy from GitHub → pilih repo ini
   - Railway otomatis deteksi Go dan deploy
   - PORT sudah otomatis diambil dari environment Railway

3. Setelah deploy, buka URL Railway yang diberikan

## 🏃 Jalankan Lokal (Testing)

```bash
go run main.go
```

Buka: http://localhost:3000

## 📱 Format Nomor WhatsApp

Sistem menerima dua format:
- `628123456789` (sudah format internasional)
- `081234567890` (otomatis dikonversi ke 628...)

## ✨ Fitur

| Fitur | Keterangan |
|-------|-----------|
| Create Account | Buat akun Pterodactyl (admin/member) |
| Create Server | Buat server + kirim WA ke buyer |
| List User | Tampil semua user (username + email + role) |
| List Server | Tampil semua server aktif |
| List Nest | Tampil semua nest tersedia |
| Sidebar | Bisa open/close |
| WhatsApp Auto-Send | Pakai Fonnte API setelah server dibuat |

## 📦 Nest & Egg Default

- Nest ID: **9**
- Egg ID: **29**
- Server: **5**

(Bisa diubah di bagian CONFIG di main.go)
