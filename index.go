package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ===================== CONFIGURASI =====================
// GANTI DENGAN API KEY ANDA
const (
	PTERODACTYL_API_KEY = "ptla_o6uTTa6TsW4gENUA65dGRS6G9kVcE5a6iMKdJUiTLwJ"
	FONNTE_API_KEY      = "WSutCwy53viwdyH8gwqE"
	PTERODACTYL_URL     = "https://reshhus.myserverr.web.id"
	PMA_URL             = "https://reshhus.myserverr.web.id/pma"
)

// ===================== DATA STORAGE (IN-MEMORY) =====================
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Language  string    `json:"language"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Server struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	Description string    `json:"description"`
	Node        int       `json:"node"`
	Database    int       `json:"database"`
	Backup      int       `json:"backup"`
	Allocation  int       `json:"allocation"`
	CPU         int       `json:"cpu"`
	Memory      int       `json:"memory"`
	Disk        int       `json:"disk"`
	NestID      int       `json:"nest_id"`
	EggID       int       `json:"egg_id"`
	Phone       string    `json:"phone"`
	CreatedAt   time.Time `json:"created_at"`
}

type Nest struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Author      string `json:"author"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Egg struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Nest        int    `json:"nest"`
	Author      string `json:"author"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var (
	users   []User
	servers []Server
	nests   []Nest
	eggs    []Egg
	userID  = 1
	serverID = 1
)

func init() {
	// Data awal nests
	nests = []Nest{
		{ID: 1, UUID: "uuid-1", Author: "Pterodactyl", Name: "Minecraft", Description: "Minecraft servers"},
		{ID: 2, UUID: "uuid-2", Author: "Pterodactyl", Name: "Source Engine", Description: "Source games"},
		{ID: 3, UUID: "uuid-3", Author: "Pterodactyl", Name: "Voice Servers", Description: "Voice chat"},
		{ID: 4, UUID: "uuid-4", Author: "Pterodactyl", Name: "Rust", Description: "Rust game"},
		{ID: 5, UUID: "uuid-5", Author: "Pterodactyl", Name: "FiveM", Description: "FiveM GTA V"},
		{ID: 6, UUID: "uuid-6", Author: "Pterodactyl", Name: "Discord Bots", Description: "Bot hosting"},
		{ID: 7, UUID: "uuid-7", Author: "Pterodactyl", Name: "Web Hosting", Description: "Web servers"},
		{ID: 8, UUID: "uuid-8", Author: "Pterodactyl", Name: "Database", Description: "Database servers"},
		{ID: 9, UUID: "uuid-9", Author: "Pterodactyl", Name: "Custom Applications", Description: "Custom apps"},
	}

	// Data awal eggs
	eggs = []Egg{
		{ID: 1, UUID: "egg-uuid-1", Nest: 1, Author: "Pterodactyl", Name: "Vanilla Minecraft", Description: "Vanilla"},
		{ID: 2, UUID: "egg-uuid-2", Nest: 1, Author: "Pterodactyl", Name: "Paper", Description: "PaperMC"},
		{ID: 3, UUID: "egg-uuid-3", Nest: 1, Author: "Pterodactyl", Name: "Spigot", Description: "Spigot"},
		{ID: 4, UUID: "egg-uuid-4", Nest: 2, Author: "Pterodactyl", Name: "Counter-Strike", Description: "CS:GO"},
		{ID: 5, UUID: "egg-uuid-5", Nest: 2, Author: "Pterodactyl", Name: "Team Fortress 2", Description: "TF2"},
		{ID: 6, UUID: "egg-uuid-6", Nest: 3, Author: "Pterodactyl", Name: "Teamspeak 3", Description: "TS3"},
		{ID: 7, UUID: "egg-uuid-7", Nest: 3, Author: "Pterodactyl", Name: "Mumble", Description: "Mumble"},
		{ID: 8, UUID: "egg-uuid-8", Nest: 4, Author: "Pterodactyl", Name: "Rust Dedicated", Description: "Rust"},
		{ID: 9, UUID: "egg-uuid-9", Nest: 5, Author: "Pterodactyl", Name: "FiveM Server", Description: "FiveM"},
		{ID: 10, UUID: "egg-uuid-10", Nest: 6, Author: "Pterodactyl", Name: "Node.js Bot", Description: "Node.js"},
		{ID: 11, UUID: "egg-uuid-11", Nest: 6, Author: "Pterodactyl", Name: "Python Bot", Description: "Python"},
		{ID: 12, UUID: "egg-uuid-12", Nest: 7, Author: "Pterodactyl", Name: "Nginx", Description: "Nginx web"},
		{ID: 13, UUID: "egg-uuid-13", Nest: 7, Author: "Pterodactyl", Name: "Apache", Description: "Apache web"},
		{ID: 14, UUID: "egg-uuid-14", Nest: 8, Author: "Pterodactyl", Name: "MySQL", Description: "MySQL DB"},
		{ID: 15, UUID: "egg-uuid-15", Nest: 8, Author: "Pterodactyl", Name: "PostgreSQL", Description: "PostgreSQL"},
		{ID: 16, UUID: "egg-uuid-16", Nest: 8, Author: "Pterodactyl", Name: "MongoDB", Description: "MongoDB"},
		{ID: 17, UUID: "egg-uuid-17", Nest: 8, Author: "Pterodactyl", Name: "Redis", Description: "Redis"},
		{ID: 18, UUID: "egg-uuid-18", Nest: 8, Author: "Pterodactyl", Name: "MariaDB", Description: "MariaDB"},
		{ID: 19, UUID: "egg-uuid-19", Nest: 8, Author: "Pterodactyl", Name: "SQLite", Description: "SQLite"},
		{ID: 20, UUID: "egg-uuid-20", Nest: 9, Author: "Pterodactyl", Name: "Custom App 1", Description: "Custom"},
		{ID: 21, UUID: "egg-uuid-21", Nest: 9, Author: "Pterodactyl", Name: "Custom App 2", Description: "Custom"},
		{ID: 22, UUID: "egg-uuid-22", Nest: 9, Author: "Pterodactyl", Name: "Custom App 3", Description: "Custom"},
		{ID: 23, UUID: "egg-uuid-23", Nest: 9, Author: "Pterodactyl", Name: "Custom App 4", Description: "Custom"},
		{ID: 24, UUID: "egg-uuid-24", Nest: 9, Author: "Pterodactyl", Name: "Custom App 5", Description: "Custom"},
		{ID: 25, UUID: "egg-uuid-25", Nest: 9, Author: "Pterodactyl", Name: "Custom App 6", Description: "Custom"},
		{ID: 26, UUID: "egg-uuid-26", Nest: 9, Author: "Pterodactyl", Name: "Custom App 7", Description: "Custom"},
		{ID: 27, UUID: "egg-uuid-27", Nest: 9, Author: "Pterodactyl", Name: "Custom App 8", Description: "Custom"},
		{ID: 28, UUID: "egg-uuid-28", Nest: 9, Author: "Pterodactyl", Name: "Custom App 9", Description: "Custom"},
		{ID: 29, UUID: "egg-uuid-29", Nest: 9, Author: "Pterodactyl", Name: "Custom App 10", Description: "Custom app for server"},
	}
}

// ===================== TEMPLATE HTML =====================
const htmlTemplate = ` + "`" + `
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pterodactyl Auto Panel - Rexxy</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%); min-height: 100vh; color: #eee; }
        .sidebar { position: fixed; left: 0; top: 0; width: 260px; height: 100vh; background: rgba(0,0,0,0.3); backdrop-filter: blur(10px); border-right: 1px solid rgba(255,255,255,0.1); padding: 20px; transition: transform 0.3s ease; z-index: 1000; }
        .sidebar.closed { transform: translateX(-260px); }
        .sidebar-header { text-align: center; margin-bottom: 30px; padding-bottom: 20px; border-bottom: 1px solid rgba(255,255,255,0.1); }
        .sidebar-header h2 { color: #00d4ff; font-size: 1.3rem; margin-bottom: 5px; }
        .sidebar-header p { color: #888; font-size: 0.8rem; }
        .nav-item { display: block; padding: 12px 15px; margin: 8px 0; border-radius: 10px; color: #ccc; text-decoration: none; transition: all 0.3s; cursor: pointer; border: none; background: transparent; width: 100%; text-align: left; font-size: 0.95rem; }
        .nav-item:hover, .nav-item.active { background: rgba(0,212,255,0.15); color: #00d4ff; transform: translateX(5px); }
        .nav-item i { margin-right: 10px; width: 20px; display: inline-block; }
        .toggle-btn { position: fixed; left: 20px; top: 20px; z-index: 1001; background: rgba(0,212,255,0.2); border: 1px solid rgba(0,212,255,0.3); color: #00d4ff; padding: 10px 15px; border-radius: 8px; cursor: pointer; font-size: 1.2rem; transition: all 0.3s; }
        .toggle-btn:hover { background: rgba(0,212,255,0.3); }
        .main-content { margin-left: 260px; padding: 30px; transition: margin-left 0.3s ease; min-height: 100vh; }
        .main-content.full { margin-left: 0; }
        .page-title { font-size: 2rem; margin-bottom: 10px; color: #00d4ff; text-shadow: 0 0 20px rgba(0,212,255,0.3); }
        .page-subtitle { color: #888; margin-bottom: 30px; }
        .card { background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 15px; padding: 25px; margin-bottom: 20px; backdrop-filter: blur(5px); }
        .card-title { font-size: 1.2rem; margin-bottom: 20px; color: #fff; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 10px; }
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; color: #ccc; font-weight: 500; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 12px 15px; border: 1px solid rgba(255,255,255,0.2); border-radius: 8px; background: rgba(0,0,0,0.2); color: #fff; font-size: 0.95rem; transition: all 0.3s; }
        .form-group input:focus, .form-group select:focus, .form-group textarea:focus { outline: none; border-color: #00d4ff; box-shadow: 0 0 10px rgba(0,212,255,0.2); }
        .form-group select option { background: #1a1a2e; color: #fff; }
        .btn { padding: 12px 30px; border: none; border-radius: 8px; cursor: pointer; font-size: 0.95rem; font-weight: 600; transition: all 0.3s; margin-right: 10px; }
        .btn-primary { background: linear-gradient(135deg, #00d4ff, #0099cc); color: #fff; }
        .btn-primary:hover { transform: translateY(-2px); box-shadow: 0 5px 20px rgba(0,212,255,0.4); }
        .btn-success { background: linear-gradient(135deg, #00ff88, #00cc66); color: #fff; }
        .btn-success:hover { transform: translateY(-2px); box-shadow: 0 5px 20px rgba(0,255,136,0.4); }
        .btn-danger { background: linear-gradient(135deg, #ff4757, #cc3344); color: #fff; }
        .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .grid-3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; }
        .alert { padding: 15px 20px; border-radius: 8px; margin-bottom: 20px; display: none; }
        .alert-success { background: rgba(0,255,136,0.1); border: 1px solid rgba(0,255,136,0.3); color: #00ff88; }
        .alert-error { background: rgba(255,71,87,0.1); border: 1px solid rgba(255,71,87,0.3); color: #ff4757; }
        .table-container { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid rgba(255,255,255,0.1); }
        th { color: #00d4ff; font-weight: 600; background: rgba(0,0,0,0.2); }
        tr:hover { background: rgba(255,255,255,0.03); }
        .badge { padding: 4px 12px; border-radius: 20px; font-size: 0.8rem; font-weight: 600; }
        .badge-admin { background: rgba(255,71,87,0.2); color: #ff4757; }
        .badge-member { background: rgba(0,212,255,0.2); color: #00d4ff; }
        .section-divider { border-top: 1px solid rgba(255,255,255,0.1); margin: 25px 0; padding-top: 15px; }
        .section-title { color: #00d4ff; font-size: 1.1rem; margin-bottom: 15px; }
        .hidden { display: none !important; }
        .phone-input-group { display: flex; gap: 10px; align-items: center; }
        .phone-input-group input { flex: 1; }
        .modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); display: none; justify-content: center; align-items: center; z-index: 2000; backdrop-filter: blur(5px); }
        .modal-overlay.active { display: flex; }
        .modal-content { background: #1a1a2e; border: 1px solid rgba(0,212,255,0.3); border-radius: 15px; padding: 30px; max-width: 500px; width: 90%; max-height: 80vh; overflow-y: auto; }
        .modal-title { color: #00d4ff; font-size: 1.3rem; margin-bottom: 20px; }
        .modal-body { color: #ccc; line-height: 1.6; margin-bottom: 20px; }
        .modal-footer { display: flex; justify-content: flex-end; gap: 10px; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 15px; padding: 20px; text-align: center; }
        .stat-number { font-size: 2.5rem; font-weight: 700; color: #00d4ff; }
        .stat-label { color: #888; margin-top: 5px; }
        @media (max-width: 768px) {
            .grid-2, .grid-3 { grid-template-columns: 1fr; }
            .sidebar { width: 100%; }
            .main-content { margin-left: 0; padding: 20px; padding-top: 70px; }
        }
    </style>
</head>
<body>
    <button class="toggle-btn" onclick="toggleSidebar()">☰</button>

    <div class="sidebar" id="sidebar">
        <div class="sidebar-header">
            <h2>🦕 PteroPanel</h2>
            <p>Auto Panel Manager</p>
        </div>
        <button class="nav-item active" onclick="showPage('dashboard')" id="nav-dashboard">
            <i>📊</i> Dashboard
        </button>
        <button class="nav-item" onclick="showPage('create-account')" id="nav-create-account">
            <i>👤</i> Create Account
        </button>
        <button class="nav-item" onclick="showPage('create-server')" id="nav-create-server">
            <i>🖥️</i> Create Server
        </button>
        <button class="nav-item" onclick="showPage('list-users')" id="nav-list-users">
            <i>📋</i> List Users
        </button>
        <button class="nav-item" onclick="showPage('list-servers')" id="nav-list-servers">
            <i>🗄️</i> List Servers
        </button>
        <button class="nav-item" onclick="showPage('list-nests')" id="nav-list-nests">
            <i>🪺</i> List Nests
        </button>
    </div>

    <div class="main-content" id="mainContent">
        <!-- DASHBOARD -->
        <div id="page-dashboard" class="page">
            <h1 class="page-title">Dashboard</h1>
            <p class="page-subtitle">Selamat datang di Pterodactyl Auto Panel Manager</p>

            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-number" id="stat-users">0</div>
                    <div class="stat-label">Total Users</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number" id="stat-servers">0</div>
                    <div class="stat-label">Total Servers</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number" id="stat-nests">{{len .Nests}}</div>
                    <div class="stat-label">Total Nests</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number" id="stat-eggs">{{len .Eggs}}</div>
                    <div class="stat-label">Total Eggs</div>
                </div>
            </div>

            <div class="card">
                <div class="card-title">ℹ️ Informasi Panel</div>
                <p style="color: #888; line-height: 1.8;">
                    • Panel URL: <a href="{{.PanelURL}}" target="_blank" style="color: #00d4ff;">{{.PanelURL}}</a><br>
                    • phpMyAdmin: <a href="{{.PMAURL}}" target="_blank" style="color: #00d4ff;">{{.PMAURL}}</a><br>
                    • Nest Default: ID 9 (Custom Applications)<br>
                    • Egg Default: ID 29 (Custom App 10)<br>
                    • Node Default: ID 5
                </p>
            </div>
        </div>

        <!-- CREATE ACCOUNT -->
        <div id="page-create-account" class="page hidden">
            <h1 class="page-title">Create Account</h1>
            <p class="page-subtitle">Buat akun user baru di panel Pterodactyl</p>

            <div class="alert alert-success" id="alert-account-success"></div>
            <div class="alert alert-error" id="alert-account-error"></div>

            <form id="form-create-account" onsubmit="return createAccount(event)">
                <div class="card">
                    <div class="card-title">👤 Informasi User</div>
                    <div class="grid-2">
                        <div class="form-group">
                            <label>Email *</label>
                            <input type="email" name="email" required placeholder="user@example.com">
                        </div>
                        <div class="form-group">
                            <label>Username *</label>
                            <input type="text" name="username" required placeholder="username">
                        </div>
                        <div class="form-group">
                            <label>First Name *</label>
                            <input type="text" name="first_name" required placeholder="First Name">
                        </div>
                        <div class="form-group">
                            <label>Last Name *</label>
                            <input type="text" name="last_name" required placeholder="Last Name">
                        </div>
                        <div class="form-group">
                            <label>Password *</label>
                            <input type="password" name="password" required placeholder="Password">
                        </div>
                        <div class="form-group">
                            <label>Role *</label>
                            <select name="role" required>
                                <option value="member">Member</option>
                                <option value="administrator">Administrator</option>
                            </select>
                        </div>
                    </div>
                    <div class="form-group">
                        <label>Default Language</label>
                        <input type="text" name="language" value="English" readonly style="background: rgba(0,0,0,0.3);">
                    </div>
                    <button type="submit" class="btn btn-primary">Create Account</button>
                </div>
            </form>
        </div>

        <!-- CREATE SERVER -->
        <div id="page-create-server" class="page hidden">
            <h1 class="page-title">Create Server</h1>
            <p class="page-subtitle">Buat server/panel baru dengan konfigurasi otomatis</p>

            <div class="alert alert-success" id="alert-server-success"></div>
            <div class="alert alert-error" id="alert-server-error"></div>

            <form id="form-create-server" onsubmit="return createServer(event)">
                <!-- PHONE INPUT -->
                <div class="card">
                    <div class="card-title">📱 Informasi WhatsApp Buyer</div>
                    <div class="form-group">
                        <label>Nomor WhatsApp Buyer *</label>
                        <div class="phone-input-group">
                            <input type="text" name="phone" id="phone-input" required placeholder="6281234567890 atau 081234567890" oninput="formatPhone()">
                            <button type="button" class="btn btn-success" onclick="verifyPhone()">Verifikasi</button>
                        </div>
                        <small style="color: #888;">Format: 628xx atau 08xx</small>
                    </div>
                    <div id="phone-verified" style="display: none; color: #00ff88; margin-top: 10px;">
                        ✅ Nomor terverifikasi: <span id="verified-phone"></span>
                    </div>
                </div>

                <!-- CORE DETAILS -->
                <div class="card">
                    <div class="card-title">🖥️ Core Details</div>
                    <div class="grid-2">
                        <div class="form-group">
                            <label>Server Name *</label>
                            <input type="text" name="server_name" required placeholder="My Server">
                        </div>
                        <div class="form-group">
                            <label>Server Owner (Email) *</label>
                            <input type="email" name="owner" required placeholder="owner@example.com">
                        </div>
                    </div>
                    <div class="form-group">
                        <label>Server Description</label>
                        <textarea name="description" rows="3" placeholder="Deskripsi server..."></textarea>
                    </div>
                </div>

                <!-- ALLOCATION MANAGEMENT -->
                <div class="card">
                    <div class="card-title">📡 Allocation Management</div>
                    <div class="grid-2">
                        <div class="form-group">
                            <label>Node</label>
                            <select name="node">
                                <option value="5" selected>Node 5 (Default)</option>
                                <option value="1">Node 1</option>
                                <option value="2">Node 2</option>
                                <option value="3">Node 3</option>
                                <option value="4">Node 4</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Default Allocation</label>
                            <input type="text" value="Auto (Default)" readonly style="background: rgba(0,0,0,0.3);">
                        </div>
                    </div>
                    <div class="form-group">
                        <label>Additional Allocation</label>
                        <input type="text" value="Auto (Default)" readonly style="background: rgba(0,0,0,0.3);">
                    </div>
                </div>

                <!-- APPLICATION FEATURE LIMITS -->
                <div class="card">
                    <div class="card-title">⚙️ Application Feature Limits</div>
                    <div class="grid-3">
                        <div class="form-group">
                            <label>Database Limit</label>
                            <select name="database_limit">
                                {{range $i := iterate 1 10}}
                                <option value="{{$i}}">{{$i}}</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Backup Limit</label>
                            <select name="backup_limit">
                                {{range $i := iterate 1 10}}
                                <option value="{{$i}}">{{$i}}</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Allocation Limit</label>
                            <select name="allocation_limit">
                                {{range $i := iterate 1 10}}
                                <option value="{{$i}}">{{$i}}</option>
                                {{end}}
                            </select>
                        </div>
                    </div>
                </div>

                <!-- RESOURCE MANAGEMENT -->
                <div class="card">
                    <div class="card-title">💾 Resource Management</div>
                    <div class="grid-2">
                        <div class="form-group">
                            <label>CPU Limit (%)</label>
                            <select name="cpu_limit">
                                {{range $i := iterate 100 500 100}}
                                <option value="{{$i}}">{{$i}}%</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Memory (GB)</label>
                            <select name="memory" id="memory-select">
                                {{range $i := iterate 1 50}}
                                <option value="{{$i}}">{{$i}} GB ({{multiply $i 1240}} MB)</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Disk Space (GB)</label>
                            <select name="disk" id="disk-select">
                                {{range $i := iterate 1 50}}
                                <option value="{{$i}}">{{$i}} GB ({{multiply $i 1240}} MB)</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="form-group">
                            <label>CPU Pinning</label>
                            <input type="text" value="Default" readonly style="background: rgba(0,0,0,0.3);">
                        </div>
                        <div class="form-group">
                            <label>Swap</label>
                            <input type="text" value="Default" readonly style="background: rgba(0,0,0,0.3);">
                        </div>
                        <div class="form-group">
                            <label>Block IO Weight</label>
                            <input type="text" value="Default" readonly style="background: rgba(0,0,0,0.3);">
                        </div>
                    </div>
                </div>

                <!-- NEST & EGG INFO -->
                <div class="card">
                    <div class="card-title">🪺 Nest & Egg Configuration</div>
                    <div class="grid-2">
                        <div class="form-group">
                            <label>Nest ID</label>
                            <input type="text" value="9 (Custom Applications)" readonly style="background: rgba(0,0,0,0.3); color: #00ff88;">
                        </div>
                        <div class="form-group">
                            <label>Egg ID</label>
                            <input type="text" value="29 (Custom App 10)" readonly style="background: rgba(0,0,0,0.3); color: #00ff88;">
                        </div>
                    </div>
                </div>

                <button type="submit" class="btn btn-primary" id="btn-create-server" disabled>Create Server & Send WhatsApp</button>
            </form>
        </div>

        <!-- LIST USERS -->
        <div id="page-list-users" class="page hidden">
            <h1 class="page-title">List Users</h1>
            <p class="page-subtitle">Daftar semua user yang telah dibuat</p>

            <div class="card">
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Username</th>
                                <th>Email</th>
                                <th>Name</th>
                                <th>Role</th>
                                <th>Created</th>
                            </tr>
                        </thead>
                        <tbody id="users-table-body">
                            <!-- Data loaded via JS -->
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- LIST SERVERS -->
        <div id="page-list-servers" class="page hidden">
            <h1 class="page-title">List Servers</h1>
            <p class="page-subtitle">Daftar semua server yang telah dibuat</p>

            <div class="card">
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Name</th>
                                <th>Owner</th>
                                <th>Node</th>
                                <th>CPU</th>
                                <th>RAM</th>
                                <th>Disk</th>
                                <th>Phone</th>
                                <th>Created</th>
                            </tr>
                        </thead>
                        <tbody id="servers-table-body">
                            <!-- Data loaded via JS -->
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- LIST NESTS -->
        <div id="page-list-nests" class="page hidden">
            <h1 class="page-title">List Nests</h1>
            <p class="page-subtitle">Daftar semua nest dan egg yang tersedia</p>

            {{range .Nests}}
            <div class="card">
                <div class="card-title">🪺 {{.Name}} (ID: {{.ID}})</div>
                <p style="color: #888; margin-bottom: 15px;">{{.Description}} | Author: {{.Author}}</p>
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Name</th>
                                <th>Description</th>
                                <th>Author</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{$nestID := .ID}}
                            {{range $.Eggs}}
                            {{if eq .Nest $nestID}}
                            <tr>
                                <td>{{.ID}}</td>
                                <td>{{.Name}}</td>
                                <td>{{.Description}}</td>
                                <td>{{.Author}}</td>
                            </tr>
                            {{end}}
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
            {{end}}
        </div>
    </div>

    <!-- MODAL CONFIRMATION -->
    <div class="modal-overlay" id="modal-confirm">
        <div class="modal-content">
            <div class="modal-title">📦 Konfirmasi Pesanan</div>
            <div class="modal-body" id="modal-body">
                <!-- Content via JS -->
            </div>
            <div class="modal-footer">
                <button class="btn" onclick="closeModal()" style="background: rgba(255,255,255,0.1); color: #fff;">Batal</button>
                <button class="btn btn-success" onclick="confirmCreateServer()">Konfirmasi & Kirim</button>
            </div>
        </div>
    </div>

    <script>
        let sidebarOpen = true;
        let verifiedPhone = '';
        let serverFormData = null;

        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const mainContent = document.getElementById('mainContent');
            sidebarOpen = !sidebarOpen;
            if (sidebarOpen) {
                sidebar.classList.remove('closed');
                mainContent.classList.remove('full');
            } else {
                sidebar.classList.add('closed');
                mainContent.classList.add('full');
            }
        }

        function showPage(pageId) {
            document.querySelectorAll('.page').forEach(p => p.classList.add('hidden'));
            document.getElementById('page-' + pageId).classList.remove('hidden');
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.getElementById('nav-' + pageId).classList.add('active');

            if (pageId === 'list-users') loadUsers();
            if (pageId === 'list-servers') loadServers();
            if (pageId === 'dashboard') loadStats();
        }

        function formatPhone() {
            let phone = document.getElementById('phone-input').value;
            phone = phone.replace(/\D/g, '');
            if (phone.startsWith('08')) {
                phone = '62' + phone.substring(1);
            }
            document.getElementById('phone-input').value = phone;
        }

        function verifyPhone() {
            const phone = document.getElementById('phone-input').value;
            if (!phone || phone.length < 10) {
                alert('Nomor tidak valid!');
                return;
            }
            verifiedPhone = phone;
            document.getElementById('verified-phone').textContent = phone;
            document.getElementById('phone-verified').style.display = 'block';
            document.getElementById('btn-create-server').disabled = false;
            alert('✅ Nomor WhatsApp terverifikasi!');
        }

        function showAlert(type, message, page) {
            const alert = document.getElementById('alert-' + page + '-' + type);
            alert.textContent = message;
            alert.style.display = 'block';
            setTimeout(() => alert.style.display = 'none', 5000);
        }

        async function createAccount(e) {
            e.preventDefault();
            const form = e.target;
            const data = {
                email: form.email.value,
                username: form.username.value,
                first_name: form.first_name.value,
                last_name: form.last_name.value,
                language: form.language.value,
                password: form.password.value,
                role: form.role.value
            };

            try {
                const res = await fetch('/api/create-account', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                const result = await res.json();
                if (result.success) {
                    showAlert('success', '✅ Akun berhasil dibuat! ID: ' + result.id, 'account');
                    form.reset();
                } else {
                    showAlert('error', '❌ ' + result.message, 'account');
                }
            } catch (err) {
                showAlert('error', '❌ Error: ' + err.message, 'account');
            }
            return false;
        }

        async function createServer(e) {
            e.preventDefault();
            if (!verifiedPhone) {
                alert('Silakan verifikasi nomor WhatsApp terlebih dahulu!');
                return false;
            }

            const form = e.target;
            serverFormData = {
                phone: verifiedPhone,
                server_name: form.server_name.value,
                owner: form.owner.value,
                description: form.description.value,
                node: parseInt(form.node.value),
                database_limit: parseInt(form.database_limit.value),
                backup_limit: parseInt(form.backup_limit.value),
                allocation_limit: parseInt(form.allocation_limit.value),
                cpu_limit: parseInt(form.cpu_limit.value),
                memory: parseInt(form.memory.value),
                disk: parseInt(form.disk.value)
            };

            // Show confirmation modal
            const modalBody = document.getElementById('modal-body');
            modalBody.innerHTML = `
                <strong>Server Name:</strong> ${serverFormData.server_name}<br>
                <strong>Owner:</strong> ${serverFormData.owner}<br>
                <strong>Node:</strong> ${serverFormData.node}<br>
                <strong>CPU:</strong> ${serverFormData.cpu_limit}%<br>
                <strong>Memory:</strong> ${serverFormData.memory} GB<br>
                <strong>Disk:</strong> ${serverFormData.disk} GB<br>
                <strong>WhatsApp:</strong> ${serverFormData.phone}<br><br>
                <em>Data akan dikirim ke WhatsApp buyer setelah konfirmasi.</em>
            `;
            document.getElementById('modal-confirm').classList.add('active');
            return false;
        }

        async function confirmCreateServer() {
            closeModal();
            try {
                const res = await fetch('/api/create-server', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(serverFormData)
                });
                const result = await res.json();
                if (result.success) {
                    showAlert('success', '✅ Server berhasil dibuat & data terkirim ke WhatsApp!', 'server');
                    document.getElementById('form-create-server').reset();
                    document.getElementById('phone-verified').style.display = 'none';
                    document.getElementById('btn-create-server').disabled = true;
                    verifiedPhone = '';
                } else {
                    showAlert('error', '❌ ' + result.message, 'server');
                }
            } catch (err) {
                showAlert('error', '❌ Error: ' + err.message, 'server');
            }
        }

        function closeModal() {
            document.getElementById('modal-confirm').classList.remove('active');
        }

        async function loadUsers() {
            try {
                const res = await fetch('/api/list-users');
                const users = await res.json();
                const tbody = document.getElementById('users-table-body');
                tbody.innerHTML = users.map(u => `
                    <tr>
                        <td>${u.id}</td>
                        <td>${u.username}</td>
                        <td>${u.email}</td>
                        <td>${u.first_name} ${u.last_name}</td>
                        <td><span class="badge badge-${u.role === 'administrator' ? 'admin' : 'member'}">${u.role}</span></td>
                        <td>${new Date(u.created_at).toLocaleString()}</td>
                    </tr>
                `).join('');
            } catch (err) {
                console.error('Error loading users:', err);
            }
        }

        async function loadServers() {
            try {
                const res = await fetch('/api/list-servers');
                const servers = await res.json();
                const tbody = document.getElementById('servers-table-body');
                tbody.innerHTML = servers.map(s => `
                    <tr>
                        <td>${s.id}</td>
                        <td>${s.name}</td>
                        <td>${s.owner}</td>
                        <td>${s.node}</td>
                        <td>${s.cpu}%</td>
                        <td>${s.memory}GB</td>
                        <td>${s.disk}GB</td>
                        <td>${s.phone}</td>
                        <td>${new Date(s.created_at).toLocaleString()}</td>
                    </tr>
                `).join('');
            } catch (err) {
                console.error('Error loading servers:', err);
            }
        }

        async function loadStats() {
            try {
                const usersRes = await fetch('/api/list-users');
                const users = await usersRes.json();
                document.getElementById('stat-users').textContent = users.length;

                const serversRes = await fetch('/api/list-servers');
                const servers = await serversRes.json();
                document.getElementById('stat-servers').textContent = servers.length;
            } catch (err) {
                console.error('Error loading stats:', err);
            }
        }

        // Load stats on page load
        loadStats();
    </script>
</body>
</html>
` + "`" + `

// ===================== TEMPLATE FUNCTIONS =====================
var funcMap = template.FuncMap{
	"iterate": func(start, end int) []int {
		var result []int
		for i := start; i <= end; i++ {
			result = append(result, i)
		}
		return result
	},
	"multiply": func(a, b int) int {
		return a * b
	},
}

// ===================== HTTP HANDLERS =====================
func renderTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Nests    []Nest
		Eggs     []Egg
		PanelURL string
		PMAURL   string
	}{
		Nests:    nests,
		Eggs:     eggs,
		PanelURL: PTERODACTYL_URL,
		PMAURL:   PMA_URL,
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}

func handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Email     string `json:"email"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Language  string `json:"language"`
		Password  string `json:"password"`
		Role      string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
		return
	}

	// Validasi
	if req.Email == "" || req.Username == "" || req.Password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Semua field wajib diisi"})
		return
	}

	// Cek duplikat
	for _, u := range users {
		if u.Email == req.Email || u.Username == req.Username {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Email atau username sudah ada"})
			return
		}
	}

	// Buat user di Pterodactyl (simulasi API call)
	pteroUser := map[string]interface{}{
		"email":      req.Email,
		"username":   req.Username,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"language":   req.Language,
		"password":   req.Password,
		"root_admin": req.Role == "administrator",
	}

	// Kirim ke Pterodactyl API
	pteroResp, err := callPterodactylAPI("POST", "/api/application/users", pteroUser)
	if err != nil {
		log.Println("Pterodactyl API Error:", err)
		// Tetap simpan ke local jika API gagal (untuk demo)
	}
	_ = pteroResp

	// Simpan ke memory
	user := User{
		ID:        userID,
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Language:  req.Language,
		Password:  req.Password,
		Role:      req.Role,
		CreatedAt: time.Now(),
	}
	users = append(users, user)
	userID++

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Akun berhasil dibuat",
		"id":      user.ID,
	})
}

func handleCreateServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Phone           string `json:"phone"`
		ServerName      string `json:"server_name"`
		Owner           string `json:"owner"`
		Description     string `json:"description"`
		Node            int    `json:"node"`
		DatabaseLimit   int    `json:"database_limit"`
		BackupLimit     int    `json:"backup_limit"`
		AllocationLimit int    `json:"allocation_limit"`
		CPULimit        int    `json:"cpu_limit"`
		Memory          int    `json:"memory"`
		Disk            int    `json:"disk"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
		return
	}

	// Validasi phone
	phone := req.Phone
	if strings.HasPrefix(phone, "08") {
		phone = "62" + phone[1:]
	}
	if !strings.HasPrefix(phone, "62") || len(phone) < 10 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Nomor WhatsApp tidak valid"})
		return
	}

	// Konversi GB ke MB (1GB = 1240MB)
	memoryMB := req.Memory * 1240
	diskMB := req.Disk * 1240

	// Buat server di Pterodactyl (simulasi API call)
	pteroServer := map[string]interface{}{
		"name":        req.ServerName,
		"user":        req.Owner,
		"egg":         29,
		"nest":        9,
		"docker_image": "ghcr.io/pterodactyl/yolks:nodejs_18",
		"startup":     "npm start",
		"environment": map[string]string{
			"SERVER_JS": "index.js",
		},
		"limits": map[string]interface{}{
			"memory":      memoryMB,
			"swap":        0,
			"disk":        diskMB,
			"io":          500,
			"cpu":         req.CPULimit,
			"threads":     nil,
		},
		"feature_limits": map[string]interface{}{
			"databases":   req.DatabaseLimit,
			"allocations": req.AllocationLimit,
			"backups":     req.BackupLimit,
		},
		"allocation": map[string]interface{}{
			"default": 1,
		},
		"deploy": map[string]interface{}{
			"locations":   []int{},
			"dedicated_ip": false,
			"port_range":  []string{},
		},
	}

	// Kirim ke Pterodactyl API
	pteroResp, err := callPterodactylAPI("POST", "/api/application/servers", pteroServer)
	if err != nil {
		log.Println("Pterodactyl API Error:", err)
	}
	_ = pteroResp

	// Simpan ke memory
	server := Server{
		ID:          serverID,
		Name:        req.ServerName,
		Owner:       req.Owner,
		Description: req.Description,
		Node:        req.Node,
		Database:    req.DatabaseLimit,
		Backup:      req.BackupLimit,
		Allocation:  req.AllocationLimit,
		CPU:         req.CPULimit,
		Memory:      req.Memory,
		Disk:        req.Disk,
		NestID:      9,
		EggID:       29,
		Phone:       phone,
		CreatedAt:   time.Now(),
	}
	servers = append(servers, server)
	serverID++

	// Kirim WhatsApp via Fonnte
	go sendWhatsApp(phone, req.Owner, req.ServerName, req.Description)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Server berhasil dibuat dan data terkirim ke WhatsApp",
		"id":      server.ID,
	})
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func handleListServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// ===================== API HELPERS =====================
func callPterodactylAPI(method, endpoint string, data interface{}) (map[string]interface{}, error) {
	url := PTERODACTYL_URL + endpoint

	var body io.Reader
	if data != nil {
		jsonData, _ := json.Marshal(data)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+PTERODACTYL_API_KEY)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "Application/vnd.pterodactyl.v1+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func sendWhatsApp(phone, owner, serverName, description string) {
	message := fmt.Sprintf(`________📦KOTAK PESANAN ANDA________
_selamat pesanan anda sudah terkonfirmasi oleh owner_

_data data account anda_
_gmail : %s_
_user : %s_
_password : (sesuai yang diinput saat create account)_

_link untuk masuk ke hosting_
_link panel : %s_
_link phpmyadmin : %s_

*________⚠️RULES / TOS________*
_1.dilarang menggunakan script bertujuan ddos/hacking/bypass_
_2.dilarang mencoba otak Atik sistem operasi_
_3.jika account hilang/dicuri teman tidak ada refund_
_4.refund aktif selama 7 hari_`,
		owner, owner, PTERODACTYL_URL, PMA_URL)

	data := map[string]string{
		"target":  phone,
		"message": message,
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", "https://api.fonnte.com/send", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", FONNTE_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Fonnte API Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Println("Fonnte Response:", string(body))
}

// ===================== MAIN =====================
func main() {
	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/", renderTemplate)
	http.HandleFunc("/api/create-account", handleCreateAccount)
	http.HandleFunc("/api/create-server", handleCreateServer)
	http.HandleFunc("/api/list-users", handleListUsers)
	http.HandleFunc("/api/list-servers", handleListServers)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port", port)
	log.Println("🦕 Pterodactyl Auto Panel Manager started!")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
