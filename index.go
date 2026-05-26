package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// ===================== CONFIG =====================
const (
	PterodactylURL    = "https://reshhus.myserverr.web.id"
	PterodactylAPIKey = "ptla_o6uTTa6TsW4gENUA65dGRS6G9kVcE5a6iMKdJUiTLwJ"
	FonnteAPIKey      = "WSutCwy53viwdyH8gwqE"
	PanelLink         = "https://reshhus.myserverr.web.id"
	PhpMyAdminLink    = "https://reshhus.myserverr.web.id/pma"
	DefaultNestID     = 9
	DefaultEggID      = 29
	DefaultServerID   = 5
	Port              = "3000"
)

// ===================== STRUCTS =====================

type CreateAccountRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
	Role      string `json:"role"`
}

type CreateServerRequest struct {
	ServerName        string `json:"server_name"`
	OwnerEmail        string `json:"owner_email"`
	Description       string `json:"description"`
	NodeID            int    `json:"node_id"`
	DatabaseLimit     int    `json:"database_limit"`
	BackupLimit       int    `json:"backup_limit"`
	AllocationLimit   int    `json:"allocation_limit"`
	CPULimit          int    `json:"cpu_limit"`
	MemoryGB          int    `json:"memory_gb"`
	DiskGB            int    `json:"disk_gb"`
	PhoneNumber       string `json:"phone_number"`
	OwnerPassword     string `json:"owner_password"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PteroUser struct {
	Attributes struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Admin    bool   `json:"root_admin"`
	} `json:"attributes"`
}

type PteroServer struct {
	Attributes struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		UUID        string `json:"uuid"`
		User        int    `json:"user"`
	} `json:"attributes"`
}

type PteroNest struct {
	Attributes struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"attributes"`
}

type PteroNode struct {
	Attributes struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"attributes"`
}

type PteroAllocation struct {
	Attributes struct {
		ID       int    `json:"id"`
		Port     int    `json:"port"`
		Assigned bool   `json:"assigned"`
		IP       string `json:"ip"`
	} `json:"attributes"`
}

// ===================== PTERODACTYL API =====================

func pteroRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, PterodactylURL+"/api/application"+endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+PterodactylAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func createPteroAccount(req CreateAccountRequest) (map[string]interface{}, error) {
	isAdmin := req.Role == "administrator"
	payload := map[string]interface{}{
		"email":        req.Email,
		"username":     req.Username,
		"first_name":   req.FirstName,
		"last_name":    req.LastName,
		"language":     "en",
		"root_admin":   isAdmin,
		"password":     req.Password,
	}

	resp, err := pteroRequest("POST", "/users", payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func getUserByEmail(email string) (int, error) {
	resp, err := pteroRequest("GET", "/users?filter[email]="+url.QueryEscape(email), nil)
	if err != nil {
		return 0, err
	}

	var result struct {
		Data []PteroUser `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	if len(result.Data) == 0 {
		return 0, fmt.Errorf("user dengan email %s tidak ditemukan", email)
	}
	return result.Data[0].Attributes.ID, nil
}

func getNodes() ([]PteroNode, error) {
	resp, err := pteroRequest("GET", "/nodes", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []PteroNode `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func getFirstFreeAllocation(nodeID int) (int, error) {
	resp, err := pteroRequest("GET", fmt.Sprintf("/nodes/%d/allocations?per_page=100", nodeID), nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Data []PteroAllocation `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	for _, a := range result.Data {
		if !a.Attributes.Assigned {
			return a.Attributes.ID, nil
		}
	}
	return 0, fmt.Errorf("tidak ada alokasi port yang tersedia di node %d", nodeID)
}

func getEggStartup(nestID, eggID int) (map[string]interface{}, error) {
	resp, err := pteroRequest("GET", fmt.Sprintf("/nests/%d/eggs/%d?include=variables", nestID, eggID), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func createPteroServer(req CreateServerRequest) (map[string]interface{}, error) {
	userID, err := getUserByEmail(req.OwnerEmail)
	if err != nil {
		return nil, fmt.Errorf("gagal cari user: %v", err)
	}

	allocID, err := getFirstFreeAllocation(req.NodeID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil alokasi: %v", err)
	}

	eggData, err := getEggStartup(DefaultNestID, DefaultEggID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil egg: %v", err)
	}

	startup := ""
	dockerImg := ""
	if attrs, ok := eggData["attributes"].(map[string]interface{}); ok {
		if s, ok := attrs["startup"].(string); ok {
			startup = s
		}
		if di, ok := attrs["docker_image"].(string); ok {
			dockerImg = di
		}
	}

	envVars := map[string]interface{}{}
	if attrs, ok := eggData["attributes"].(map[string]interface{}); ok {
		if rels, ok := attrs["relationships"].(map[string]interface{}); ok {
			if vars, ok := rels["variables"].(map[string]interface{}); ok {
				if data, ok := vars["data"].([]interface{}); ok {
					for _, v := range data {
						if varMap, ok := v.(map[string]interface{}); ok {
							if varAttrs, ok := varMap["attributes"].(map[string]interface{}); ok {
								envKey := fmt.Sprintf("%v", varAttrs["env_variable"])
								envDefault := fmt.Sprintf("%v", varAttrs["default_value"])
								envVars[envKey] = envDefault
							}
						}
					}
				}
			}
		}
	}

	memoryMB := req.MemoryGB * 1024
	diskMB := req.DiskGB * 1024

	payload := map[string]interface{}{
		"name":         req.ServerName,
		"user":         userID,
		"egg":          DefaultEggID,
		"docker_image": dockerImg,
		"startup":      startup,
		"environment":  envVars,
		"limits": map[string]interface{}{
			"memory":  memoryMB,
			"swap":    0,
			"disk":    diskMB,
			"io":      500,
			"cpu":     req.CPULimit,
		},
		"feature_limits": map[string]interface{}{
			"databases":   req.DatabaseLimit,
			"backups":     req.BackupLimit,
			"allocations": req.AllocationLimit,
		},
		"allocation": map[string]interface{}{
			"default": allocID,
		},
		"description": req.Description,
		"start_on_completion": false,
	}

	resp, err := pteroRequest("POST", "/servers", payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func listUsers() ([]PteroUser, error) {
	resp, err := pteroRequest("GET", "/users?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []PteroUser `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func listServers() ([]PteroServer, error) {
	resp, err := pteroRequest("GET", "/servers?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []PteroServer `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func listNests() ([]PteroNest, error) {
	resp, err := pteroRequest("GET", "/nests", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []PteroNest `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ===================== FONNTE WHATSAPP =====================

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "08") {
		phone = "628" + phone[2:]
	}
	return phone
}

func sendWhatsApp(phone, message string) error {
	phone = normalizePhone(phone)

	data := url.Values{}
	data.Set("target", phone)
	data.Set("message", message)
	data.Set("countryCode", "62")

	req, err := http.NewRequest("POST", "https://api.fonnte.com/send", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", FonnteAPIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Fonnte response: %s", string(body))
	return nil
}

func buildWhatsAppMessage(email, username, password string) string {
	return fmt.Sprintf(`________📦KOTAK PESANAN ANDA________
_selamat pesanan anda sudah terkonfirmasi oleh owner_

_data data account anda_
_gmail : %s_
_user : %s_
_password : %s_

_link untuk masuk ke hosting_
_link panel : %s_
_link phpmyadmin : %s_

*________⚠️RULES / TOS________*
_1.dilarang menggunakan script bertujuan ddos/hacking/bypass_
_2.dilarang mencoba otak Atik sistem operasi_
_3.jika account hilang/dicuri teman tidak ada refund_
_4.refund aktif selama 7 hari_`,
		email, username, password, PanelLink, PhpMyAdminLink)
}

// ===================== HTTP HANDLERS =====================

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid JSON: " + err.Error()})
		return
	}
	if req.Email == "" || req.Username == "" || req.Password == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Email, username, dan password wajib diisi"})
		return
	}

	result, err := createPteroAccount(req)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Gagal buat akun: " + err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: "Akun berhasil dibuat!", Data: result})
}

func handleCreateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid JSON: " + err.Error()})
		return
	}
	if req.ServerName == "" || req.OwnerEmail == "" || req.PhoneNumber == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Nama server, email owner, dan nomor HP wajib diisi"})
		return
	}
	if req.NodeID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Node ID wajib dipilih"})
		return
	}

	// Set defaults
	if req.DatabaseLimit == 0 { req.DatabaseLimit = 1 }
	if req.BackupLimit == 0 { req.BackupLimit = 1 }
	if req.AllocationLimit == 0 { req.AllocationLimit = 1 }
	if req.CPULimit == 0 { req.CPULimit = 100 }
	if req.MemoryGB == 0 { req.MemoryGB = 1 }
	if req.DiskGB == 0 { req.DiskGB = 5 }

	result, err := createPteroServer(req)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Gagal buat server: " + err.Error()})
		return
	}

	// Send WhatsApp
	msg := buildWhatsAppMessage(req.OwnerEmail, req.OwnerEmail, req.OwnerPassword)
	if err := sendWhatsApp(req.PhoneNumber, msg); err != nil {
		log.Printf("Gagal kirim WhatsApp: %v", err)
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Server berhasil dibuat & notifikasi WhatsApp dikirim ke %s!", normalizePhone(req.PhoneNumber)),
		Data:    result,
	})
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := listUsers()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: users})
}

func handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := listServers()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: servers})
}

func handleListNests(w http.ResponseWriter, r *http.Request) {
	nests, err := listNests()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: nests})
}

func handleGetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := getNodes()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: nodes})
}

// ===================== HTML TEMPLATE =====================

const htmlTemplate = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1.0"/>
<title>RenzyDev Panel Manager</title>
<link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet"/>
<style>
:root {
  --bg: #050a0e;
  --surface: #0d1117;
  --surface2: #161b22;
  --surface3: #21262d;
  --border: #30363d;
  --accent: #00d4ff;
  --accent2: #7c3aed;
  --accent3: #06b6d4;
  --green: #3fb950;
  --red: #f85149;
  --yellow: #d29922;
  --text: #e6edf3;
  --text2: #8b949e;
  --text3: #6e7681;
  --sidebar-w: 260px;
  --glow: 0 0 20px rgba(0,212,255,0.15);
}
*{margin:0;padding:0;box-sizing:border-box;}
body{font-family:'Space Grotesk',sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex;overflow-x:hidden;}
/* Sidebar */
#sidebar{
  width:var(--sidebar-w);min-width:var(--sidebar-w);height:100vh;position:fixed;left:0;top:0;
  background:var(--surface);border-right:1px solid var(--border);
  display:flex;flex-direction:column;z-index:100;
  transition:transform 0.3s cubic-bezier(0.4,0,0.2,1);
}
#sidebar.closed{transform:translateX(calc(-1 * var(--sidebar-w)));}
.sidebar-header{
  padding:24px 20px;border-bottom:1px solid var(--border);
  display:flex;align-items:center;gap:12px;
}
.logo-icon{
  width:36px;height:36px;border-radius:8px;
  background:linear-gradient(135deg,var(--accent2),var(--accent));
  display:flex;align-items:center;justify-content:center;
  font-size:16px;font-weight:700;color:#fff;
  box-shadow:0 0 15px rgba(124,58,237,0.4);
  flex-shrink:0;
}
.logo-text{font-size:14px;font-weight:700;color:var(--text);}
.logo-sub{font-size:11px;color:var(--text3);font-family:'JetBrains Mono',monospace;}
.nav{padding:16px 0;flex:1;}
.nav-section{padding:8px 20px 4px;font-size:10px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:1.5px;}
.nav-item{
  display:flex;align-items:center;gap:12px;padding:10px 20px;margin:2px 8px;
  border-radius:8px;cursor:pointer;transition:all 0.2s;
  font-size:14px;color:var(--text2);font-weight:500;
}
.nav-item:hover{background:var(--surface2);color:var(--text);}
.nav-item.active{background:linear-gradient(135deg,rgba(124,58,237,0.2),rgba(0,212,255,0.1));color:var(--accent);border:1px solid rgba(0,212,255,0.2);}
.nav-item .icon{width:18px;text-align:center;font-size:16px;}
.sidebar-footer{padding:16px;border-top:1px solid var(--border);}
.status-badge{
  display:flex;align-items:center;gap:8px;padding:8px 12px;
  background:var(--surface2);border-radius:8px;border:1px solid var(--border);
  font-size:12px;color:var(--text2);
}
.status-dot{width:8px;height:8px;border-radius:50%;background:var(--green);box-shadow:0 0 6px var(--green);animation:pulse 2s infinite;}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:0.5}}
/* Toggle Button */
#toggleBtn{
  position:fixed;left:16px;top:16px;z-index:200;
  width:40px;height:40px;border-radius:10px;
  background:var(--surface2);border:1px solid var(--border);
  cursor:pointer;display:flex;align-items:center;justify-content:center;
  transition:all 0.2s;color:var(--text);font-size:18px;
}
#toggleBtn:hover{background:var(--surface3);border-color:var(--accent);color:var(--accent);}
/* Main Content */
#main{
  margin-left:var(--sidebar-w);flex:1;min-height:100vh;
  transition:margin-left 0.3s cubic-bezier(0.4,0,0.2,1);
  padding:24px;padding-top:72px;
}
#main.expanded{margin-left:0;}
.page{display:none;animation:fadeIn 0.3s ease;}
.page.active{display:block;}
@keyframes fadeIn{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}
/* Page Header */
.page-header{margin-bottom:32px;}
.page-title{font-size:26px;font-weight:700;color:var(--text);margin-bottom:4px;}
.page-title span{color:var(--accent);}
.page-sub{font-size:14px;color:var(--text3);}
/* Cards */
.card{
  background:var(--surface);border:1px solid var(--border);
  border-radius:12px;overflow:hidden;
}
.card-header{
  padding:20px 24px;border-bottom:1px solid var(--border);
  display:flex;align-items:center;gap:10px;
}
.card-header h3{font-size:15px;font-weight:600;color:var(--text);}
.card-body{padding:24px;}
/* Form */
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:16px;}
.form-group{display:flex;flex-direction:column;gap:6px;}
.form-group.full{grid-column:1/-1;}
label{font-size:12px;font-weight:600;color:var(--text2);text-transform:uppercase;letter-spacing:0.5px;}
input,select,textarea{
  background:var(--surface2);border:1px solid var(--border);
  border-radius:8px;padding:10px 14px;
  font-family:'Space Grotesk',sans-serif;font-size:14px;color:var(--text);
  transition:all 0.2s;outline:none;width:100%;
}
input:focus,select:focus,textarea:focus{
  border-color:var(--accent);
  box-shadow:0 0 0 3px rgba(0,212,255,0.1);
}
textarea{resize:vertical;min-height:80px;}
select option{background:var(--surface2);}
/* Section divider */
.section-divider{
  display:flex;align-items:center;gap:12px;margin:24px 0 16px;
  font-size:11px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:1px;
}
.section-divider::before,.section-divider::after{content:'';flex:1;height:1px;background:var(--border);}
/* Buttons */
.btn{
  display:inline-flex;align-items:center;justify-content:center;gap:8px;
  padding:10px 20px;border-radius:8px;font-family:'Space Grotesk',sans-serif;
  font-size:14px;font-weight:600;cursor:pointer;transition:all 0.2s;border:none;
}
.btn-primary{
  background:linear-gradient(135deg,var(--accent2),var(--accent3));
  color:#fff;box-shadow:0 4px 15px rgba(124,58,237,0.3);
}
.btn-primary:hover{transform:translateY(-1px);box-shadow:0 6px 20px rgba(124,58,237,0.4);}
.btn-primary:active{transform:translateY(0);}
.btn-primary:disabled{opacity:0.5;cursor:not-allowed;transform:none;}
.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text2);}
.btn-outline:hover{border-color:var(--accent);color:var(--accent);}
.form-actions{margin-top:24px;display:flex;gap:12px;justify-content:flex-end;}
/* Alert */
.alert{
  padding:14px 16px;border-radius:8px;margin-bottom:16px;
  font-size:14px;display:flex;align-items:flex-start;gap:10px;
}
.alert-success{background:rgba(63,185,80,0.1);border:1px solid rgba(63,185,80,0.3);color:var(--green);}
.alert-error{background:rgba(248,81,73,0.1);border:1px solid rgba(248,81,73,0.3);color:var(--red);}
.alert-info{background:rgba(0,212,255,0.1);border:1px solid rgba(0,212,255,0.2);color:var(--accent);}
/* Table */
.table-wrap{overflow-x:auto;}
table{width:100%;border-collapse:collapse;font-size:14px;}
thead tr{border-bottom:2px solid var(--border);}
th{padding:12px 16px;text-align:left;font-size:11px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:0.5px;}
tbody tr{border-bottom:1px solid var(--border);transition:background 0.15s;}
tbody tr:hover{background:var(--surface2);}
td{padding:12px 16px;color:var(--text2);}
td strong{color:var(--text);}
.badge{
  display:inline-flex;align-items:center;padding:3px 10px;
  border-radius:20px;font-size:11px;font-weight:600;
}
.badge-admin{background:rgba(124,58,237,0.2);color:#a78bfa;border:1px solid rgba(124,58,237,0.3);}
.badge-member{background:rgba(0,212,255,0.1);color:var(--accent3);border:1px solid rgba(0,212,255,0.2);}
.badge-server{background:rgba(63,185,80,0.1);color:var(--green);border:1px solid rgba(63,185,80,0.3);}
/* Loading */
.loading{display:flex;align-items:center;gap:8px;color:var(--text3);font-size:14px;padding:40px 0;}
.spinner{width:16px;height:16px;border:2px solid var(--border);border-top-color:var(--accent);border-radius:50%;animation:spin 0.6s linear infinite;}
@keyframes spin{to{transform:rotate(360deg)}}
/* Empty state */
.empty{text-align:center;padding:60px 20px;color:var(--text3);}
.empty .empty-icon{font-size:48px;margin-bottom:12px;}
.empty p{font-size:14px;}
/* Phone section */
.phone-section{
  background:linear-gradient(135deg,rgba(0,212,255,0.05),rgba(124,58,237,0.05));
  border:1px solid rgba(0,212,255,0.2);border-radius:12px;padding:20px;margin-bottom:20px;
}
.phone-section h4{font-size:14px;font-weight:600;color:var(--accent);margin-bottom:12px;display:flex;align-items:center;gap:8px;}
/* Toast */
#toast{
  position:fixed;bottom:24px;right:24px;z-index:1000;
  display:flex;flex-direction:column;gap:8px;
}
.toast-item{
  padding:12px 16px;border-radius:10px;font-size:13px;font-weight:500;
  min-width:280px;max-width:360px;
  display:flex;align-items:center;gap:10px;
  animation:slideIn 0.3s ease;box-shadow:0 8px 24px rgba(0,0,0,0.4);
}
.toast-success{background:#1a3d2e;border:1px solid rgba(63,185,80,0.4);color:#3fb950;}
.toast-error{background:#3d1a1a;border:1px solid rgba(248,81,73,0.4);color:#f85149;}
@keyframes slideIn{from{opacity:0;transform:translateX(20px)}to{opacity:1;transform:translateX(0)}}
/* Refresh btn */
.refresh-btn{
  width:32px;height:32px;border-radius:8px;background:var(--surface2);
  border:1px solid var(--border);cursor:pointer;display:flex;align-items:center;
  justify-content:center;color:var(--text2);font-size:14px;transition:all 0.2s;margin-left:auto;
}
.refresh-btn:hover{border-color:var(--accent);color:var(--accent);}
/* responsive */
@media(max-width:768px){
  .form-grid{grid-template-columns:1fr;}
  #main{padding:16px;padding-top:64px;}
  .page-title{font-size:20px;}
}
</style>
</head>
<body>

<!-- Sidebar Toggle -->
<button id="toggleBtn" onclick="toggleSidebar()" title="Toggle Sidebar">☰</button>

<!-- Sidebar -->
<aside id="sidebar">
  <div class="sidebar-header">
    <div class="logo-icon">R</div>
    <div>
      <div class="logo-text">RenzyDev Panel</div>
      <div class="logo-sub">v1.0.0 · Manager</div>
    </div>
  </div>
  <nav class="nav">
    <div class="nav-section">Manajemen</div>
    <div class="nav-item active" onclick="showPage('create-account')">
      <span class="icon">👤</span> Buat Akun
    </div>
    <div class="nav-item" onclick="showPage('create-server')">
      <span class="icon">🖥️</span> Buat Server
    </div>
    <div class="nav-section">Data</div>
    <div class="nav-item" onclick="showPage('list-users');loadUsers()">
      <span class="icon">👥</span> List User
    </div>
    <div class="nav-item" onclick="showPage('list-servers');loadServers()">
      <span class="icon">📦</span> List Server
    </div>
    <div class="nav-item" onclick="showPage('list-nests');loadNests()">
      <span class="icon">🥚</span> List Nest
    </div>
  </nav>
  <div class="sidebar-footer">
    <div class="status-badge">
      <div class="status-dot"></div>
      <span>Panel Terhubung</span>
    </div>
  </div>
</aside>

<!-- Main -->
<main id="main">

  <!-- CREATE ACCOUNT -->
  <div id="page-create-account" class="page active">
    <div class="page-header">
      <div class="page-title">Buat <span>Akun</span></div>
      <div class="page-sub">Buat akun baru di panel Pterodactyl</div>
    </div>
    <div id="alert-account"></div>
    <div class="card">
      <div class="card-header">
        <span>👤</span>
        <h3>Informasi Akun</h3>
      </div>
      <div class="card-body">
        <div class="form-grid">
          <div class="form-group">
            <label>Email</label>
            <input type="email" id="acc-email" placeholder="contoh@gmail.com"/>
          </div>
          <div class="form-group">
            <label>Username</label>
            <input type="text" id="acc-username" placeholder="username"/>
          </div>
          <div class="form-group">
            <label>First Name</label>
            <input type="text" id="acc-firstname" placeholder="Nama depan"/>
          </div>
          <div class="form-group">
            <label>Last Name</label>
            <input type="text" id="acc-lastname" placeholder="Nama belakang"/>
          </div>
          <div class="form-group">
            <label>Password</label>
            <input type="password" id="acc-password" placeholder="Password kuat"/>
          </div>
          <div class="form-group">
            <label>Role</label>
            <select id="acc-role">
              <option value="member">Member</option>
              <option value="administrator">Administrator</option>
            </select>
          </div>
          <div class="form-group">
            <label>Default Language</label>
            <input type="text" value="English" disabled style="opacity:0.5"/>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="resetForm('account')">Reset</button>
          <button class="btn btn-primary" id="btn-create-account" onclick="createAccount()">
            ✨ Buat Akun
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- CREATE SERVER -->
  <div id="page-create-server" class="page">
    <div class="page-header">
      <div class="page-title">Buat <span>Server</span></div>
      <div class="page-sub">Provisioning server baru di Pterodactyl · Nest ID: {{.NestID}} · Egg ID: {{.EggID}}</div>
    </div>
    <div id="alert-server"></div>

    <!-- Phone section -->
    <div class="phone-section">
      <h4>📱 Nomor WhatsApp Buyer</h4>
      <div class="form-group">
        <label>Nomor HP (format 628xxx atau 08xxx)</label>
        <input type="text" id="srv-phone" placeholder="628123456789 atau 081234567890"/>
      </div>
      <div style="margin-top:8px;font-size:12px;color:var(--text3)">⚠️ Notifikasi order akan dikirim ke nomor ini setelah server berhasil dibuat</div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="card-header"><span>📋</span><h3>Core Details</h3></div>
      <div class="card-body">
        <div class="form-grid">
          <div class="form-group">
            <label>Server Name</label>
            <input type="text" id="srv-name" placeholder="Nama server"/>
          </div>
          <div class="form-group">
            <label>Server Owner (Email)</label>
            <input type="email" id="srv-email" placeholder="email@gmail.com"/>
          </div>
          <div class="form-group">
            <label>Password Owner</label>
            <input type="password" id="srv-owner-pass" placeholder="Password untuk notif WA"/>
          </div>
          <div class="form-group full">
            <label>Deskripsi Server</label>
            <textarea id="srv-desc" placeholder="Deskripsi server..."></textarea>
          </div>
        </div>
      </div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="card-header"><span>🌐</span><h3>Allocation Management</h3></div>
      <div class="card-body">
        <div class="form-grid">
          <div class="form-group full">
            <label>Node</label>
            <select id="srv-node">
              <option value="">Loading nodes...</option>
            </select>
          </div>
        </div>
        <div style="margin-top:12px;font-size:12px;color:var(--text3)">📌 Default & Additional allocation dipilih otomatis dari node yang tersedia</div>
      </div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="card-header"><span>⚙️</span><h3>Feature Limits</h3></div>
      <div class="card-body">
        <div class="form-grid">
          <div class="form-group">
            <label>Database Limit</label>
            <select id="srv-db">
              <option value="1">1</option><option value="2">2</option><option value="3">3</option>
              <option value="4">4</option><option value="5">5</option><option value="6">6</option>
              <option value="7">7</option><option value="8">8</option><option value="9">9</option>
              <option value="10">10</option>
            </select>
          </div>
          <div class="form-group">
            <label>Backup Limit</label>
            <select id="srv-backup">
              <option value="1">1</option><option value="2">2</option><option value="3">3</option>
              <option value="4">4</option><option value="5">5</option><option value="6">6</option>
              <option value="7">7</option><option value="8">8</option><option value="9">9</option>
              <option value="10">10</option>
            </select>
          </div>
          <div class="form-group">
            <label>Allocation Limit</label>
            <select id="srv-alloc-limit">
              <option value="1">1</option><option value="2">2</option><option value="3">3</option>
              <option value="4">4</option><option value="5">5</option><option value="6">6</option>
              <option value="7">7</option><option value="8">8</option><option value="9">9</option>
              <option value="10">10</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <div class="card">
      <div class="card-header"><span>💾</span><h3>Resource Management</h3></div>
      <div class="card-body">
        <div class="form-grid">
          <div class="form-group">
            <label>CPU Limit</label>
            <select id="srv-cpu">
              <option value="100">100% (1 Core)</option>
              <option value="200">200% (2 Core)</option>
              <option value="300">300% (3 Core)</option>
              <option value="400">400% (4 Core)</option>
              <option value="500">500% (5 Core)</option>
            </select>
          </div>
          <div class="form-group">
            <label>Memory</label>
            <select id="srv-memory">
              <option value="1">1 GB (1024 MB)</option>
              <option value="2">2 GB (2048 MB)</option>
              <option value="4">4 GB (4096 MB)</option>
              <option value="8">8 GB (8192 MB)</option>
              <option value="16">16 GB (16384 MB)</option>
              <option value="32">32 GB (32768 MB)</option>
              <option value="50">50 GB (51200 MB)</option>
            </select>
          </div>
          <div class="form-group">
            <label>Disk Space</label>
            <select id="srv-disk">
              <option value="5">5 GB</option>
              <option value="10">10 GB</option>
              <option value="20">20 GB</option>
              <option value="30">30 GB</option>
              <option value="50">50 GB</option>
            </select>
          </div>
          <div class="form-group">
            <label>CPU Pinning</label>
            <input type="text" value="Default (otomatis)" disabled style="opacity:0.5"/>
          </div>
          <div class="form-group">
            <label>Swap</label>
            <input type="text" value="Default (0)" disabled style="opacity:0.5"/>
          </div>
          <div class="form-group">
            <label>Block IO Weight</label>
            <input type="text" value="Default (500)" disabled style="opacity:0.5"/>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="resetForm('server')">Reset</button>
          <button class="btn btn-primary" id="btn-create-server" onclick="createServer()">
            🚀 Buat Server & Kirim WA
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- LIST USERS -->
  <div id="page-list-users" class="page">
    <div class="page-header">
      <div class="page-title">List <span>User</span></div>
      <div class="page-sub">Semua pengguna terdaftar di panel</div>
    </div>
    <div class="card">
      <div class="card-header">
        <span>👥</span><h3>Daftar User</h3>
        <button class="refresh-btn" onclick="loadUsers()" title="Refresh">↻</button>
      </div>
      <div class="card-body" id="users-content">
        <div class="loading"><div class="spinner"></div>Loading...</div>
      </div>
    </div>
  </div>

  <!-- LIST SERVERS -->
  <div id="page-list-servers" class="page">
    <div class="page-header">
      <div class="page-title">List <span>Server</span></div>
      <div class="page-sub">Semua server yang berjalan di panel</div>
    </div>
    <div class="card">
      <div class="card-header">
        <span>📦</span><h3>Daftar Server</h3>
        <button class="refresh-btn" onclick="loadServers()" title="Refresh">↻</button>
      </div>
      <div class="card-body" id="servers-content">
        <div class="loading"><div class="spinner"></div>Loading...</div>
      </div>
    </div>
  </div>

  <!-- LIST NESTS -->
  <div id="page-list-nests" class="page">
    <div class="page-header">
      <div class="page-title">List <span>Nest</span></div>
      <div class="page-sub">Semua nest yang tersedia di panel</div>
    </div>
    <div class="card">
      <div class="card-header">
        <span>🥚</span><h3>Daftar Nest</h3>
        <button class="refresh-btn" onclick="loadNests()" title="Refresh">↻</button>
      </div>
      <div class="card-body" id="nests-content">
        <div class="loading"><div class="spinner"></div>Loading...</div>
      </div>
    </div>
  </div>

</main>

<!-- Toast Container -->
<div id="toast"></div>

<script>
// ===================== SIDEBAR =====================
let sidebarOpen = true;
function toggleSidebar(){
  sidebarOpen = !sidebarOpen;
  document.getElementById('sidebar').classList.toggle('closed', !sidebarOpen);
  document.getElementById('main').classList.toggle('expanded', !sidebarOpen);
  document.getElementById('toggleBtn').style.left = sidebarOpen ? '16px' : '16px';
}

// ===================== PAGES =====================
function showPage(page) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
  document.getElementById('page-' + page).classList.add('active');
  event.currentTarget.classList.add('active');
}

// ===================== TOAST =====================
function showToast(msg, type='success'){
  const toast = document.getElementById('toast');
  const el = document.createElement('div');
  el.className = 'toast-item toast-' + type;
  el.innerHTML = (type==='success'?'✅':'❌') + ' ' + msg;
  toast.appendChild(el);
  setTimeout(()=>el.remove(), 4000);
}

// ===================== ALERT =====================
function showAlert(containerId, msg, type='success'){
  const el = document.getElementById(containerId);
  el.innerHTML = '<div class="alert alert-'+type+'"><span>'+(type==='success'?'✅':type==='error'?'❌':'ℹ️')+'</span><span>'+msg+'</span></div>';
  setTimeout(()=>el.innerHTML='', 6000);
}

// ===================== RESET =====================
function resetForm(type){
  if(type==='account'){
    ['acc-email','acc-username','acc-firstname','acc-lastname','acc-password'].forEach(id=>{
      document.getElementById(id).value='';
    });
    document.getElementById('acc-role').value='member';
  } else {
    ['srv-name','srv-email','srv-desc','srv-phone','srv-owner-pass'].forEach(id=>{
      document.getElementById(id).value='';
    });
  }
}

// ===================== LOAD NODES =====================
async function loadNodes(){
  try {
    const res = await fetch('/api/nodes');
    const data = await res.json();
    const sel = document.getElementById('srv-node');
    sel.innerHTML = '';
    if(data.data && data.data.length > 0){
      data.data.forEach(n => {
        const opt = document.createElement('option');
        opt.value = n.attributes.id;
        opt.textContent = n.attributes.name + ' (ID: ' + n.attributes.id + ')';
        sel.appendChild(opt);
      });
    } else {
      sel.innerHTML = '<option value="">Tidak ada node</option>';
    }
  } catch(e) {
    document.getElementById('srv-node').innerHTML = '<option value="">Gagal load nodes</option>';
  }
}

// ===================== CREATE ACCOUNT =====================
async function createAccount(){
  const btn = document.getElementById('btn-create-account');
  const email = document.getElementById('acc-email').value.trim();
  const username = document.getElementById('acc-username').value.trim();
  const firstname = document.getElementById('acc-firstname').value.trim();
  const lastname = document.getElementById('acc-lastname').value.trim();
  const password = document.getElementById('acc-password').value;
  const role = document.getElementById('acc-role').value;

  if(!email || !username || !password){
    showAlert('alert-account', 'Email, username, dan password wajib diisi!', 'error');
    return;
  }

  btn.disabled = true;
  btn.textContent = '⏳ Membuat akun...';

  try {
    const res = await fetch('/api/create-account', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({email, username, first_name:firstname, last_name:lastname, password, role})
    });
    const data = await res.json();
    if(data.success){
      showAlert('alert-account', '✅ Akun berhasil dibuat untuk ' + email, 'success');
      showToast('Akun ' + username + ' berhasil dibuat!');
      resetForm('account');
    } else {
      showAlert('alert-account', 'Gagal: ' + data.message, 'error');
      showToast('Gagal buat akun: ' + data.message, 'error');
    }
  } catch(e) {
    showAlert('alert-account', 'Error: ' + e.message, 'error');
  }

  btn.disabled = false;
  btn.innerHTML = '✨ Buat Akun';
}

// ===================== CREATE SERVER =====================
async function createServer(){
  const btn = document.getElementById('btn-create-server');
  const phone = document.getElementById('srv-phone').value.trim();
  const name = document.getElementById('srv-name').value.trim();
  const email = document.getElementById('srv-email').value.trim();
  const desc = document.getElementById('srv-desc').value.trim();
  const nodeId = parseInt(document.getElementById('srv-node').value);
  const ownerPass = document.getElementById('srv-owner-pass').value;

  if(!phone){
    showAlert('alert-server', '⚠️ Nomor WhatsApp buyer wajib diisi terlebih dahulu!', 'error');
    document.getElementById('srv-phone').focus();
    return;
  }
  if(!name || !email){
    showAlert('alert-server', 'Nama server dan email owner wajib diisi!', 'error');
    return;
  }
  if(!nodeId){
    showAlert('alert-server', 'Silakan pilih Node!', 'error');
    return;
  }

  btn.disabled = true;
  btn.textContent = '⏳ Membuat server...';

  const payload = {
    server_name: name,
    owner_email: email,
    description: desc,
    node_id: nodeId,
    database_limit: parseInt(document.getElementById('srv-db').value),
    backup_limit: parseInt(document.getElementById('srv-backup').value),
    allocation_limit: parseInt(document.getElementById('srv-alloc-limit').value),
    cpu_limit: parseInt(document.getElementById('srv-cpu').value),
    memory_gb: parseInt(document.getElementById('srv-memory').value),
    disk_gb: parseInt(document.getElementById('srv-disk').value),
    phone_number: phone,
    owner_password: ownerPass,
  };

  try {
    const res = await fetch('/api/create-server', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if(data.success){
      showAlert('alert-server', '🚀 ' + data.message, 'success');
      showToast('Server berhasil dibuat & WA terkirim!');
      resetForm('server');
    } else {
      showAlert('alert-server', 'Gagal: ' + data.message, 'error');
      showToast('Gagal buat server', 'error');
    }
  } catch(e) {
    showAlert('alert-server', 'Error: ' + e.message, 'error');
    showToast('Error: ' + e.message, 'error');
  }

  btn.disabled = false;
  btn.innerHTML = '🚀 Buat Server & Kirim WA';
}

// ===================== LOAD USERS =====================
async function loadUsers(){
  document.getElementById('users-content').innerHTML = '<div class="loading"><div class="spinner"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-users');
    const data = await res.json();
    if(!data.success){ throw new Error(data.message); }
    const users = data.data || [];
    if(users.length === 0){
      document.getElementById('users-content').innerHTML = '<div class="empty"><div class="empty-icon">👥</div><p>Belum ada user</p></div>';
      return;
    }
    let html = '<div class="table-wrap"><table><thead><tr><th>#</th><th>Username</th><th>Email</th><th>Role</th></tr></thead><tbody>';
    users.forEach((u,i) => {
      const attrs = u.attributes;
      const isAdmin = attrs.root_admin;
      html += '<tr><td>'+attrs.id+'</td><td><strong>'+attrs.username+'</strong></td><td>'+attrs.email+'</td><td><span class="badge '+(isAdmin?'badge-admin':'badge-member')+'">'+(isAdmin?'Admin':'Member')+'</span></td></tr>';
    });
    html += '</tbody></table></div>';
    document.getElementById('users-content').innerHTML = html;
  } catch(e) {
    document.getElementById('users-content').innerHTML = '<div class="alert alert-error">❌ Gagal load users: ' + e.message + '</div>';
  }
}

// ===================== LOAD SERVERS =====================
async function loadServers(){
  document.getElementById('servers-content').innerHTML = '<div class="loading"><div class="spinner"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-servers');
    const data = await res.json();
    if(!data.success){ throw new Error(data.message); }
    const servers = data.data || [];
    if(servers.length === 0){
      document.getElementById('servers-content').innerHTML = '<div class="empty"><div class="empty-icon">📦</div><p>Belum ada server</p></div>';
      return;
    }
    let html = '<div class="table-wrap"><table><thead><tr><th>#</th><th>Nama</th><th>Deskripsi</th><th>UUID</th><th>Status</th></tr></thead><tbody>';
    servers.forEach(s => {
      const a = s.attributes;
      html += '<tr><td>'+a.id+'</td><td><strong>'+a.name+'</strong></td><td>'+(a.description||'-')+'</td><td><code style="font-family:\'JetBrains Mono\';font-size:11px;color:var(--text3)">'+a.uuid.substring(0,8)+'...</code></td><td><span class="badge badge-server">Active</span></td></tr>';
    });
    html += '</tbody></table></div>';
    document.getElementById('servers-content').innerHTML = html;
  } catch(e) {
    document.getElementById('servers-content').innerHTML = '<div class="alert alert-error">❌ Gagal load servers: ' + e.message + '</div>';
  }
}

// ===================== LOAD NESTS =====================
async function loadNests(){
  document.getElementById('nests-content').innerHTML = '<div class="loading"><div class="spinner"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-nests');
    const data = await res.json();
    if(!data.success){ throw new Error(data.message); }
    const nests = data.data || [];
    if(nests.length === 0){
      document.getElementById('nests-content').innerHTML = '<div class="empty"><div class="empty-icon">🥚</div><p>Belum ada nest</p></div>';
      return;
    }
    let html = '<div class="table-wrap"><table><thead><tr><th>ID</th><th>Nama</th></tr></thead><tbody>';
    nests.forEach(n => {
      const a = n.attributes;
      html += '<tr><td><strong>'+a.id+'</strong></td><td>'+a.name+'</td></tr>';
    });
    html += '</tbody></table></div>';
    document.getElementById('nests-content').innerHTML = html;
  } catch(e) {
    document.getElementById('nests-content').innerHTML = '<div class="alert alert-error">❌ Gagal load nests: ' + e.message + '</div>';
  }
}

// Init
loadNodes();
</script>
</body>
</html>`

// ===================== MAIN =====================

func main() {
	// Parse template
	tmpl := template.Must(template.New("index").Parse(htmlTemplate))

	// Static page handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			NestID int
			EggID  int
		}{DefaultNestID, DefaultEggID}
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Template error: %v", err)
		}
	})

	// API Routes
	http.HandleFunc("/api/create-account", handleCreateAccount)
	http.HandleFunc("/api/create-server", handleCreateServer)
	http.HandleFunc("/api/list-users", handleListUsers)
	http.HandleFunc("/api/list-servers", handleListServers)
	http.HandleFunc("/api/list-nests", handleListNests)
	http.HandleFunc("/api/nodes", handleGetNodes)

	// Health check for Railway
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"RenzyDev Panel Manager"}`)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = Port
	}

	log.Printf("🚀 RenzyDev Panel Manager running on :%s", port)
	log.Printf("📡 Pterodactyl URL: %s", PterodactylURL)
	log.Printf("🌐 Open: http://localhost:%s", port)

	// Validate config on startup
	if strings.Contains(PterodactylAPIKey, "MASUKKAN") {
		log.Println("⚠️  PERINGATAN: API Key Pterodactyl belum diset di config!")
	}
	if strings.Contains(FonnteAPIKey, "MASUKKAN") {
		log.Println("⚠️  PERINGATAN: API Key Fonnte belum diset di config!")
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Suppress unused import warning
var _ = strconv.Itoa
