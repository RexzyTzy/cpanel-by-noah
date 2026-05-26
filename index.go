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
	"strings"
	"sync"
	"text/template"
	"time"
)

// ===================== CONFIG =====================
const (
	PterodactylURL    = "https://reshhus.myserverr.web.id"
	PterodactylAPIKey = "ptla_o6uTTa6TsW4gENUA65dGRS6G9kVcE5a6iMKdJUiTLwJ"
	FonnteAPIKey      = "WSutCwy53viwdyH8gwqE"
	PanelLink         = "https://reshhus.myserverr.web.id"
	PhpMyAdminLink    = "https://reshhus.myserverr.web.id/pma"
	Port              = "3000"
)

// ===================== SERVER HISTORY (in-memory) =====================
type HistoryEntry struct {
	ID          int       `json:"id"`
	ServerName  string    `json:"server_name"`
	OwnerEmail  string    `json:"owner_email"`
	OwnerUser   string    `json:"owner_username"`
	PhoneNumber string    `json:"phone_number"`
	CPU         int       `json:"cpu"`
	MemoryGB    int       `json:"memory_gb"`
	DiskGB      int       `json:"disk_gb"`
	CreatedAt   time.Time `json:"created_at"`
}

var (
	historyMu      sync.Mutex
	serverHistory  []HistoryEntry
	historyCounter int
)

func addHistory(e HistoryEntry) {
	historyMu.Lock()
	defer historyMu.Unlock()
	historyCounter++
	e.ID = historyCounter
	e.CreatedAt = time.Now()
	serverHistory = append([]HistoryEntry{e}, serverHistory...)
	if len(serverHistory) > 100 {
		serverHistory = serverHistory[:100]
	}
}

func getHistory() []HistoryEntry {
	historyMu.Lock()
	defer historyMu.Unlock()
	return append([]HistoryEntry{}, serverHistory...)
}

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
	ServerName      string `json:"server_name"`
	OwnerEmail      string `json:"owner_email"`
	OwnerUsername   string `json:"owner_username"`
	Description     string `json:"description"`
	NodeID          int    `json:"node_id"`
	NestID          int    `json:"nest_id"`
	EggID           int    `json:"egg_id"`
	DatabaseLimit   int    `json:"database_limit"`
	BackupLimit     int    `json:"backup_limit"`
	AllocationLimit int    `json:"allocation_limit"`
	CPULimit        int    `json:"cpu_limit"`
	MemoryGB        int    `json:"memory_gb"`
	DiskGB          int    `json:"disk_gb"`
	PhoneNumber     string `json:"phone_number"`
	OwnerPassword   string `json:"owner_password"`
}

type EditServerRequest struct {
	ServerID  int `json:"server_id"`
	CPULimit  int `json:"cpu_limit"`
	MemoryGB  int `json:"memory_gb"`
	DiskGB    int `json:"disk_gb"`
	DatabaseLimit   int `json:"database_limit"`
	BackupLimit     int `json:"backup_limit"`
	AllocationLimit int `json:"allocation_limit"`
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
		Suspended   bool   `json:"suspended"`
		Limits      struct {
			Memory int `json:"memory"`
			Disk   int `json:"disk"`
			CPU    int `json:"cpu"`
		} `json:"limits"`
		FeatureLimits struct {
			Databases   int `json:"databases"`
			Backups     int `json:"backups"`
			Allocations int `json:"allocations"`
		} `json:"feature_limits"`
	} `json:"attributes"`
}

type PteroNest struct {
	Attributes struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"attributes"`
}

type PteroEgg struct {
	Attributes struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		DockerImage string `json:"docker_image"`
		Startup     string `json:"startup"`
	} `json:"attributes"`
}

type PteroNode struct {
	Attributes struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		Memory          int    `json:"memory"`
		Disk            int    `json:"disk"`
		MemoryOveralloc int    `json:"memory_overallocate"`
		DiskOveralloc   int    `json:"disk_overallocate"`
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
func pteroRequest(method, endpoint string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, PterodactylURL+"/api/application"+endpoint, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+PterodactylAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("koneksi ke panel gagal: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		// Parse Pterodactyl error detail
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errs, ok := errResp["errors"].([]interface{}); ok && len(errs) > 0 {
				if e, ok := errs[0].(map[string]interface{}); ok {
					detail := fmt.Sprintf("%v", e["detail"])
					code := fmt.Sprintf("%v", e["code"])
					return nil, resp.StatusCode, fmt.Errorf("[%s] %s", code, detail)
				}
			}
		}
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, nil
}

func getUserByEmail(email string) (int, error) {
	resp, _, err := pteroRequest("GET", "/users?filter[email]="+url.QueryEscape(email), nil)
	if err != nil {
		return 0, err
	}
	var result struct{ Data []PteroUser `json:"data"` }
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	if len(result.Data) == 0 {
		return 0, fmt.Errorf("user dengan email '%s' tidak ditemukan di panel", email)
	}
	return result.Data[0].Attributes.ID, nil
}

func getNodes() ([]PteroNode, error) {
	resp, _, err := pteroRequest("GET", "/nodes", nil)
	if err != nil {
		return nil, err
	}
	var result struct{ Data []PteroNode `json:"data"` }
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func getFirstFreeAllocation(nodeID int) (int, error) {
	resp, _, err := pteroRequest("GET", fmt.Sprintf("/nodes/%d/allocations?per_page=100", nodeID), nil)
	if err != nil {
		return 0, err
	}
	var result struct{ Data []PteroAllocation `json:"data"` }
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}
	for _, a := range result.Data {
		if !a.Attributes.Assigned {
			return a.Attributes.ID, nil
		}
	}
	return 0, fmt.Errorf("tidak ada port kosong di node %d", nodeID)
}

func getEggStartup(nestID, eggID int) (map[string]interface{}, error) {
	resp, _, err := pteroRequest("GET", fmt.Sprintf("/nests/%d/eggs/%d?include=variables", nestID, eggID), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func createPteroAccount(req CreateAccountRequest) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"email": req.Email, "username": req.Username,
		"first_name": req.FirstName, "last_name": req.LastName,
		"language": "en", "root_admin": req.Role == "administrator",
		"password": req.Password,
	}
	resp, _, err := pteroRequest("POST", "/users", payload)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)
	return result, nil
}

func createPteroServer(req CreateServerRequest) (map[string]interface{}, error) {
	userID, err := getUserByEmail(req.OwnerEmail)
	if err != nil {
		return nil, fmt.Errorf("gagal cari user: %v", err)
	}
	allocID, err := getFirstFreeAllocation(req.NodeID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil alokasi port: %v", err)
	}
	eggData, err := getEggStartup(req.NestID, req.EggID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil egg config: %v", err)
	}

	startup, dockerImg := "", ""
	envVars := map[string]interface{}{}
	if attrs, ok := eggData["attributes"].(map[string]interface{}); ok {
		if s, ok := attrs["startup"].(string); ok { startup = s }
		if d, ok := attrs["docker_image"].(string); ok { dockerImg = d }
		if rels, ok := attrs["relationships"].(map[string]interface{}); ok {
			if vars, ok := rels["variables"].(map[string]interface{}); ok {
				if data, ok := vars["data"].([]interface{}); ok {
					for _, v := range data {
						if vm, ok := v.(map[string]interface{}); ok {
							if va, ok := vm["attributes"].(map[string]interface{}); ok {
								envVars[fmt.Sprintf("%v", va["env_variable"])] = fmt.Sprintf("%v", va["default_value"])
							}
						}
					}
				}
			}
		}
	}

	payload := map[string]interface{}{
		"name": req.ServerName, "user": userID,
		"egg": req.EggID, "docker_image": dockerImg,
		"startup": startup, "environment": envVars,
		"limits": map[string]interface{}{
			"memory": req.MemoryGB * 1024, "swap": 0,
			"disk": req.DiskGB * 1024, "io": 500, "cpu": req.CPULimit,
		},
		"feature_limits": map[string]interface{}{
			"databases": req.DatabaseLimit, "backups": req.BackupLimit,
			"allocations": req.AllocationLimit,
		},
		"allocation": map[string]interface{}{"default": allocID},
		"description": req.Description, "start_on_completion": false,
	}

	resp, _, err := pteroRequest("POST", "/servers", payload)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)
	return result, nil
}

func listUsers() ([]PteroUser, error) {
	resp, _, err := pteroRequest("GET", "/users?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var result struct{ Data []PteroUser `json:"data"` }
	json.Unmarshal(resp, &result)
	return result.Data, nil
}

func listServers() ([]PteroServer, error) {
	resp, _, err := pteroRequest("GET", "/servers?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var result struct{ Data []PteroServer `json:"data"` }
	json.Unmarshal(resp, &result)
	return result.Data, nil
}

func listNests() ([]PteroNest, error) {
	resp, _, err := pteroRequest("GET", "/nests", nil)
	if err != nil {
		return nil, err
	}
	var result struct{ Data []PteroNest `json:"data"` }
	json.Unmarshal(resp, &result)
	return result.Data, nil
}

func listEggs(nestID int) ([]PteroEgg, error) {
	resp, _, err := pteroRequest("GET", fmt.Sprintf("/nests/%d/eggs?include=variables", nestID), nil)
	if err != nil {
		return nil, err
	}
	var result struct{ Data []PteroEgg `json:"data"` }
	json.Unmarshal(resp, &result)
	return result.Data, nil
}

func deleteUser(userID int) error {
	_, _, err := pteroRequest("DELETE", fmt.Sprintf("/users/%d", userID), nil)
	return err
}

func deleteServer(serverID int) error {
	_, _, err := pteroRequest("DELETE", fmt.Sprintf("/servers/%d", serverID), nil)
	return err
}

func suspendServer(serverID int) error {
	_, _, err := pteroRequest("POST", fmt.Sprintf("/servers/%d/suspend", serverID), nil)
	return err
}

func unsuspendServer(serverID int) error {
	_, _, err := pteroRequest("POST", fmt.Sprintf("/servers/%d/unsuspend", serverID), nil)
	return err
}

func reinstallServer(serverID int) error {
	_, _, err := pteroRequest("POST", fmt.Sprintf("/servers/%d/reinstall", serverID), nil)
	return err
}

func editServerBuild(req EditServerRequest) error {
	payload := map[string]interface{}{
		"allocation": 0, // will be ignored if 0 in some versions
		"limits": map[string]interface{}{
			"memory": req.MemoryGB * 1024, "swap": 0,
			"disk": req.DiskGB * 1024, "io": 500, "cpu": req.CPULimit,
		},
		"feature_limits": map[string]interface{}{
			"databases": req.DatabaseLimit, "backups": req.BackupLimit,
			"allocations": req.AllocationLimit,
		},
	}
	_, _, err := pteroRequest("PATCH", fmt.Sprintf("/servers/%d/build", req.ServerID), payload)
	return err
}

func getServerDetail(serverID int) (PteroServer, error) {
	resp, _, err := pteroRequest("GET", fmt.Sprintf("/servers/%d", serverID), nil)
	if err != nil {
		return PteroServer{}, err
	}
	var result PteroServer
	json.Unmarshal(resp, &result)
	return result, nil
}

// ===================== FONNTE WHATSAPP =====================
func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	if strings.HasPrefix(phone, "08") {
		phone = "628" + phone[2:]
	} else if strings.HasPrefix(phone, "8") && len(phone) >= 9 {
		phone = "62" + phone
	} else if strings.HasPrefix(phone, "+62") {
		phone = phone[1:]
	}

	if !strings.HasPrefix(phone, "628") {
		return "", fmt.Errorf("format nomor tidak valid. Gunakan format 628xxx atau 08xxx (contoh: 6281234567890)")
	}
	if len(phone) < 11 || len(phone) > 15 {
		return "", fmt.Errorf("panjang nomor tidak valid (%d digit). Nomor Indonesia biasanya 10-13 digit", len(phone)-2)
	}
	return phone, nil
}

func sendWhatsApp(phone, message string) error {
	normalized, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	data := url.Values{}
	data.Set("target", normalized)
	data.Set("message", message)
	data.Set("countryCode", "62")

	req, err := http.NewRequest("POST", "https://api.fonnte.com/send", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", FonnteAPIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal koneksi ke Fonnte: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var fResp map[string]interface{}
	if json.Unmarshal(body, &fResp) == nil {
		if status, ok := fResp["status"].(bool); ok && !status {
			return fmt.Errorf("Fonnte error: %v", fResp["reason"])
		}
	}
	log.Printf("Fonnte OK → %s", normalized)
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
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "JSON tidak valid: " + err.Error()}); return
	}
	if req.Email == "" || req.Username == "" || req.Password == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Email, username, dan password wajib diisi"}); return
	}
	result, err := createPteroAccount(req)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: "Akun berhasil dibuat!", Data: result})
}

func handleCreateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "JSON tidak valid: " + err.Error()}); return
	}

	// Validasi nomor WA ketat
	if req.PhoneNumber == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Nomor WhatsApp buyer wajib diisi"}); return
	}
	normalizedPhone, err := normalizePhone(req.PhoneNumber)
	if err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Nomor WA tidak valid: " + err.Error()}); return
	}

	if req.ServerName == "" || req.OwnerEmail == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Nama server dan email owner wajib diisi"}); return
	}
	if req.NodeID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Node wajib dipilih"}); return
	}
	if req.NestID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Nest wajib dipilih"}); return
	}
	if req.EggID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Egg wajib dipilih"}); return
	}

	if req.DatabaseLimit == 0 { req.DatabaseLimit = 1 }
	if req.BackupLimit == 0 { req.BackupLimit = 1 }
	if req.AllocationLimit == 0 { req.AllocationLimit = 1 }
	if req.CPULimit == 0 { req.CPULimit = 100 }
	if req.MemoryGB == 0 { req.MemoryGB = 1 }
	if req.DiskGB == 0 { req.DiskGB = 5 }

	result, err := createPteroServer(req)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}

	// Simpan ke history
	addHistory(HistoryEntry{
		ServerName: req.ServerName, OwnerEmail: req.OwnerEmail,
		OwnerUser: req.OwnerUsername, PhoneNumber: normalizedPhone,
		CPU: req.CPULimit, MemoryGB: req.MemoryGB, DiskGB: req.DiskGB,
	})

	// Kirim WA
	msg := buildWhatsAppMessage(req.OwnerEmail, req.OwnerUsername, req.OwnerPassword)
	waErr := ""
	if err := sendWhatsApp(req.PhoneNumber, msg); err != nil {
		waErr = " (WA gagal: " + err.Error() + ")"
		log.Printf("WA error: %v", err)
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Server '%s' berhasil dibuat! Notifikasi WA dikirim ke %s%s", req.ServerName, normalizedPhone, waErr),
		Data:    result,
	})
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := listUsers()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: users})
}

func handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := listServers()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: servers})
}

func handleListNests(w http.ResponseWriter, r *http.Request) {
	nests, err := listNests()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: nests})
}

func handleListEggs(w http.ResponseWriter, r *http.Request) {
	nestIDStr := r.URL.Query().Get("nest_id")
	var nestID int
	fmt.Sscanf(nestIDStr, "%d", &nestID)
	if nestID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "nest_id wajib diisi"}); return
	}
	eggs, err := listEggs(nestID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: eggs})
}

func handleGetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := getNodes()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: nodes})
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req struct{ UserID int `json:"user_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.UserID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "user_id wajib diisi"}); return
	}
	if err := deleteUser(req.UserID); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("User ID %d berhasil dihapus", req.UserID)})
}

func handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req struct{ ServerID int `json:"server_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.ServerID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "server_id wajib diisi"}); return
	}
	if err := deleteServer(req.ServerID); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("Server ID %d berhasil dihapus", req.ServerID)})
}

func handleSuspendServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req struct {
		ServerID  int  `json:"server_id"`
		Suspended bool `json:"suspended"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.ServerID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "server_id wajib diisi"}); return
	}
	var err error
	action := "disuspend"
	if req.Suspended {
		err = suspendServer(req.ServerID)
	} else {
		err = unsuspendServer(req.ServerID)
		action = "diaktifkan"
	}
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("Server ID %d berhasil %s", req.ServerID, action)})
}

func handleReinstallServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req struct{ ServerID int `json:"server_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.ServerID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "server_id wajib diisi"}); return
	}
	if err := reinstallServer(req.ServerID); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("Server ID %d sedang diinstall ulang", req.ServerID)})
}

func handleEditServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req EditServerRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.ServerID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "server_id wajib diisi"}); return
	}
	if req.DatabaseLimit == 0 { req.DatabaseLimit = 1 }
	if req.BackupLimit == 0 { req.BackupLimit = 1 }
	if req.AllocationLimit == 0 { req.AllocationLimit = 1 }
	if req.CPULimit == 0 { req.CPULimit = 100 }
	if req.MemoryGB == 0 { req.MemoryGB = 1 }
	if req.DiskGB == 0 { req.DiskGB = 5 }
	if err := editServerBuild(req); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("Server ID %d berhasil diupdate", req.ServerID)})
}

func handleServerDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	var serverID int
	fmt.Sscanf(idStr, "%d", &serverID)
	if serverID == 0 {
		respondJSON(w, 400, APIResponse{Success: false, Message: "id wajib diisi"}); return
	}
	server, err := getServerDetail(serverID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	respondJSON(w, 200, APIResponse{Success: true, Data: server})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	users, errU := listUsers()
	servers, errS := listServers()
	nodes, errN := getNodes()

	type DashData struct {
		TotalUsers   int         `json:"total_users"`
		TotalServers int         `json:"total_servers"`
		TotalNodes   int         `json:"total_nodes"`
		Suspended    int         `json:"suspended"`
		AdminCount   int         `json:"admin_count"`
		Nodes        []PteroNode `json:"nodes"`
		Errors       []string    `json:"errors,omitempty"`
	}

	d := DashData{}
	var errs []string
	if errU != nil { errs = append(errs, "users: "+errU.Error()) } else {
		d.TotalUsers = len(users)
		for _, u := range users { if u.Attributes.Admin { d.AdminCount++ } }
	}
	if errS != nil { errs = append(errs, "servers: "+errS.Error()) } else {
		d.TotalServers = len(servers)
		for _, s := range servers { if s.Attributes.Suspended { d.Suspended++ } }
	}
	if errN != nil { errs = append(errs, "nodes: "+errN.Error()) } else {
		d.TotalNodes = len(nodes)
		d.Nodes = nodes
	}
	if len(errs) > 0 { d.Errors = errs }
	respondJSON(w, 200, APIResponse{Success: true, Data: d})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, 200, APIResponse{Success: true, Data: getHistory()})
}

func handleValidatePhone(w http.ResponseWriter, r *http.Request) {
	var req struct{ Phone string `json:"phone"` }
	json.NewDecoder(r.Body).Decode(&req)
	normalized, err := normalizePhone(req.Phone)
	if err != nil {
		respondJSON(w, 200, APIResponse{Success: false, Message: err.Error()})
		return
	}
	respondJSON(w, 200, APIResponse{Success: true, Message: normalized, Data: normalized})
}

func handleSendWA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, 405, APIResponse{Success: false, Message: "Method not allowed"}); return
	}
	var req struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Phone == "" || req.Message == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "phone dan message wajib diisi"}); return
	}
	if err := sendWhatsApp(req.Phone, req.Message); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: err.Error()}); return
	}
	normalized, _ := normalizePhone(req.Phone)
	respondJSON(w, 200, APIResponse{Success: true, Message: "WA berhasil dikirim ke " + normalized})
}

// ===================== HTML TEMPLATE =====================
const htmlPage = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1.0"/>
<title>RenzyDev Panel Manager</title>
<link href="https://fonts.googleapis.com/css2?family=Syne:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet"/>
<script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.4.0/chart.umd.min.js"></script>
<style>
:root{
  --bg:#060a0f;--surface:#0c1117;--surface2:#141a22;--surface3:#1c2330;
  --border:#252d3a;--border2:#2e3848;
  --accent:#00e5ff;--accent2:#7c3aed;--accent3:#06b6d4;
  --green:#22c55e;--red:#ef4444;--yellow:#f59e0b;--orange:#f97316;
  --text:#e2eaf4;--text2:#8b99ae;--text3:#4d5d72;
  --sidebar-w:256px;
  --radius:10px;
}
*{margin:0;padding:0;box-sizing:border-box;}
body{font-family:'Syne',sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex;overflow-x:hidden;}
/* SCROLLBAR */
::-webkit-scrollbar{width:5px;height:5px}
::-webkit-scrollbar-track{background:var(--surface)}
::-webkit-scrollbar-thumb{background:var(--border2);border-radius:99px}

/* SIDEBAR */
#sidebar{
  width:var(--sidebar-w);min-width:var(--sidebar-w);height:100vh;
  position:fixed;left:0;top:0;
  background:var(--surface);
  border-right:1px solid var(--border);
  display:flex;flex-direction:column;z-index:100;
  transition:transform 0.3s cubic-bezier(.4,0,.2,1);
}
#sidebar.closed{transform:translateX(calc(-1 * var(--sidebar-w)));}
.sb-head{
  padding:20px 16px;border-bottom:1px solid var(--border);
  display:flex;align-items:center;gap:12px;
}
.sb-logo{
  width:34px;height:34px;border-radius:9px;flex-shrink:0;
  background:linear-gradient(135deg,var(--accent2),var(--accent));
  display:flex;align-items:center;justify-content:center;
  font-size:15px;font-weight:800;color:#fff;
  box-shadow:0 0 20px rgba(124,58,237,.35);
}
.sb-title{font-size:14px;font-weight:700;line-height:1.2;}
.sb-sub{font-size:10px;color:var(--text3);font-family:'JetBrains Mono',monospace;margin-top:2px;}
.sb-nav{padding:12px 0;flex:1;overflow-y:auto;}
.sb-sec{padding:6px 16px 4px;font-size:10px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:1.5px;margin-top:8px;}
.sb-item{
  display:flex;align-items:center;gap:10px;padding:9px 16px;margin:1px 8px;
  border-radius:8px;cursor:pointer;transition:all .18s;
  font-size:13px;font-weight:500;color:var(--text2);
}
.sb-item:hover{background:var(--surface2);color:var(--text);}
.sb-item.active{
  background:linear-gradient(135deg,rgba(124,58,237,.18),rgba(0,229,255,.08));
  color:var(--accent);border:1px solid rgba(0,229,255,.18);
}
.sb-item .ic{width:17px;text-align:center;font-size:15px;}
.sb-foot{padding:12px 16px;border-top:1px solid var(--border);}
.sb-status{
  display:flex;align-items:center;gap:8px;padding:8px 12px;
  background:var(--surface2);border-radius:8px;border:1px solid var(--border);
  font-size:12px;color:var(--text2);
}
.dot-live{width:7px;height:7px;border-radius:50%;background:var(--green);box-shadow:0 0 6px var(--green);animation:pulse 2s infinite;}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}

/* TOGGLE */
#tog{
  position:fixed;left:14px;top:14px;z-index:200;
  width:38px;height:38px;border-radius:9px;
  background:var(--surface2);border:1px solid var(--border);
  cursor:pointer;display:flex;align-items:center;justify-content:center;
  transition:all .18s;color:var(--text2);font-size:17px;
}
#tog:hover{border-color:var(--accent);color:var(--accent);}

/* MAIN */
#main{
  margin-left:var(--sidebar-w);flex:1;min-height:100vh;
  transition:margin-left .3s cubic-bezier(.4,0,.2,1);
  padding:24px;padding-top:64px;max-width:1200px;
}
#main.exp{margin-left:0;}
.page{display:none;animation:fadeUp .28s ease;}
.page.active{display:block;}
@keyframes fadeUp{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:translateY(0)}}

/* PAGE HEADER */
.ph{margin-bottom:28px;}
.ph-title{font-size:24px;font-weight:800;line-height:1.2;}
.ph-title span{color:var(--accent);}
.ph-sub{font-size:13px;color:var(--text3);margin-top:3px;}

/* DASHBOARD STATS */
.stats-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:20px;}
.stat-card{
  background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  padding:18px;position:relative;overflow:hidden;
  transition:border-color .2s;
}
.stat-card:hover{border-color:var(--border2);}
.stat-card::before{
  content:'';position:absolute;top:0;left:0;right:0;height:2px;
  background:var(--stat-color,var(--accent));
}
.stat-icon{font-size:22px;margin-bottom:10px;}
.stat-val{font-size:28px;font-weight:800;color:var(--text);}
.stat-label{font-size:12px;color:var(--text3);margin-top:3px;}
.chart-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-bottom:20px;}
.chart-card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:18px;}
.chart-title{font-size:13px;font-weight:600;color:var(--text2);margin-bottom:14px;}

/* CARDS */
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);overflow:hidden;margin-bottom:14px;}
.card-h{padding:16px 20px;border-bottom:1px solid var(--border);display:flex;align-items:center;gap:9px;}
.card-h h3{font-size:14px;font-weight:600;color:var(--text);}
.card-b{padding:20px;}

/* FORM */
.fg{display:flex;flex-direction:column;gap:5px;}
.fg2{display:grid;grid-template-columns:1fr 1fr;gap:14px;}
.fg2.c3{grid-template-columns:1fr 1fr 1fr;}
.fg.full{grid-column:1/-1;}
label{font-size:11px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:.5px;}
input,select,textarea{
  background:var(--surface2);border:1px solid var(--border);
  border-radius:8px;padding:9px 13px;
  font-family:'Syne',sans-serif;font-size:13px;color:var(--text);
  transition:all .18s;outline:none;width:100%;
}
input:focus,select:focus,textarea:focus{
  border-color:var(--accent);box-shadow:0 0 0 3px rgba(0,229,255,.08);
}
input.valid{border-color:var(--green);}
input.invalid{border-color:var(--red);}
textarea{resize:vertical;min-height:72px;}
select option{background:var(--surface2);}
.hint{font-size:11px;margin-top:4px;}
.hint.ok{color:var(--green);}
.hint.err{color:var(--red);}

/* SECTION DIV */
.sdiv{
  display:flex;align-items:center;gap:10px;margin:20px 0 14px;
  font-size:10px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:1px;
}
.sdiv::before,.sdiv::after{content:'';flex:1;height:1px;background:var(--border);}

/* BUTTONS */
.btn{
  display:inline-flex;align-items:center;justify-content:center;gap:7px;
  padding:9px 18px;border-radius:8px;font-family:'Syne',sans-serif;
  font-size:13px;font-weight:600;cursor:pointer;transition:all .18s;border:none;
}
.btn-primary{
  background:linear-gradient(135deg,var(--accent2),var(--accent3));
  color:#fff;box-shadow:0 4px 14px rgba(124,58,237,.3);
}
.btn-primary:hover{transform:translateY(-1px);box-shadow:0 6px 20px rgba(124,58,237,.4);}
.btn-primary:active{transform:none;}
.btn-primary:disabled{opacity:.45;cursor:not-allowed;transform:none;}
.btn-ghost{background:transparent;border:1px solid var(--border);color:var(--text2);}
.btn-ghost:hover{border-color:var(--accent);color:var(--accent);}
.btn-danger{background:rgba(239,68,68,.12);border:1px solid rgba(239,68,68,.3);color:var(--red);}
.btn-danger:hover{background:rgba(239,68,68,.2);}
.btn-warn{background:rgba(245,158,11,.12);border:1px solid rgba(245,158,11,.3);color:var(--yellow);}
.btn-warn:hover{background:rgba(245,158,11,.2);}
.btn-info{background:rgba(0,229,255,.1);border:1px solid rgba(0,229,255,.2);color:var(--accent);}
.btn-info:hover{background:rgba(0,229,255,.18);}
.btn-sm{padding:5px 10px;font-size:11px;border-radius:6px;}
.fa{margin-top:20px;display:flex;gap:10px;justify-content:flex-end;}

/* ALERT */
.alert{padding:12px 15px;border-radius:8px;margin-bottom:12px;font-size:13px;display:flex;align-items:flex-start;gap:9px;}
.alert-s{background:rgba(34,197,94,.08);border:1px solid rgba(34,197,94,.25);color:var(--green);}
.alert-e{background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.25);color:var(--red);}
.alert-i{background:rgba(0,229,255,.08);border:1px solid rgba(0,229,255,.18);color:var(--accent);}

/* TABLE */
.tw{overflow-x:auto;}
.search-bar{margin-bottom:12px;position:relative;}
.search-bar input{padding-left:34px;}
.search-bar::before{content:'🔍';position:absolute;left:10px;top:50%;transform:translateY(-50%);font-size:13px;pointer-events:none;}
table{width:100%;border-collapse:collapse;font-size:13px;}
thead tr{border-bottom:2px solid var(--border);}
th{padding:10px 14px;text-align:left;font-size:10px;font-weight:600;color:var(--text3);text-transform:uppercase;letter-spacing:.5px;}
tbody tr{border-bottom:1px solid var(--border);transition:background .12s;}
tbody tr:hover{background:var(--surface2);}
td{padding:10px 14px;color:var(--text2);vertical-align:middle;}
td strong{color:var(--text);}
.badge{display:inline-flex;align-items:center;padding:2px 9px;border-radius:20px;font-size:10px;font-weight:600;}
.b-admin{background:rgba(124,58,237,.18);color:#a78bfa;border:1px solid rgba(124,58,237,.3);}
.b-member{background:rgba(0,229,255,.08);color:var(--accent3);border:1px solid rgba(0,229,255,.18);}
.b-active{background:rgba(34,197,94,.1);color:var(--green);border:1px solid rgba(34,197,94,.25);}
.b-suspend{background:rgba(239,68,68,.1);color:var(--red);border:1px solid rgba(239,68,68,.25);}
.act-btns{display:flex;gap:5px;flex-wrap:wrap;}

/* MODAL */
.modal-overlay{
  display:none;position:fixed;inset:0;background:rgba(0,0,0,.7);
  z-index:500;align-items:center;justify-content:center;backdrop-filter:blur(4px);
}
.modal-overlay.open{display:flex;}
.modal{
  background:var(--surface);border:1px solid var(--border2);border-radius:14px;
  padding:24px;width:90%;max-width:520px;max-height:90vh;overflow-y:auto;
  animation:fadeUp .25s ease;
}
.modal-h{display:flex;align-items:center;justify-content:space-between;margin-bottom:20px;}
.modal-h h3{font-size:16px;font-weight:700;}
.modal-close{
  width:30px;height:30px;border-radius:7px;background:var(--surface2);border:1px solid var(--border);
  cursor:pointer;display:flex;align-items:center;justify-content:center;color:var(--text2);font-size:16px;
}
.modal-close:hover{border-color:var(--red);color:var(--red);}

/* LOADING */
.loading{display:flex;align-items:center;gap:8px;color:var(--text3);font-size:13px;padding:36px 0;}
.spin{width:15px;height:15px;border:2px solid var(--border2);border-top-color:var(--accent);border-radius:50%;animation:spin .55s linear infinite;}
@keyframes spin{to{transform:rotate(360deg)}}
.empty{text-align:center;padding:52px 20px;color:var(--text3);}
.empty .ei{font-size:40px;margin-bottom:10px;}

/* PHONE INPUT */
.phone-box{
  background:linear-gradient(135deg,rgba(0,229,255,.04),rgba(124,58,237,.04));
  border:1px solid rgba(0,229,255,.15);border-radius:var(--radius);padding:16px;margin-bottom:16px;
}
.phone-box h4{font-size:13px;font-weight:600;color:var(--accent);margin-bottom:10px;display:flex;align-items:center;gap:7px;}

/* WA PREVIEW */
.wa-preview{
  background:var(--surface2);border:1px solid var(--border);border-radius:9px;
  padding:14px;font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text2);white-space:pre-wrap;word-break:break-all;line-height:1.6;
  max-height:280px;overflow-y:auto;
}
.wa-preview-wrap{position:relative;}
.copy-btn{
  position:absolute;top:8px;right:8px;
  padding:4px 9px;font-size:10px;border-radius:6px;
  background:var(--surface3);border:1px solid var(--border2);
  color:var(--text2);cursor:pointer;font-family:'Syne',sans-serif;font-weight:600;
  transition:all .18s;
}
.copy-btn:hover{border-color:var(--accent);color:var(--accent);}
.copy-btn.copied{border-color:var(--green);color:var(--green);}

/* HISTORY */
.hist-item{
  display:flex;align-items:flex-start;gap:12px;padding:12px 0;
  border-bottom:1px solid var(--border);
}
.hist-item:last-child{border-bottom:none;}
.hist-num{
  width:28px;height:28px;border-radius:7px;background:var(--surface2);
  border:1px solid var(--border);display:flex;align-items:center;justify-content:center;
  font-size:11px;font-weight:700;color:var(--text3);flex-shrink:0;
}
.hist-info{flex:1;}
.hist-name{font-size:13px;font-weight:600;color:var(--text);}
.hist-meta{font-size:11px;color:var(--text3);margin-top:3px;}
.hist-time{font-size:10px;color:var(--text3);font-family:'JetBrains Mono',monospace;margin-top:2px;}

/* TOAST */
#toast{position:fixed;bottom:22px;right:22px;z-index:1000;display:flex;flex-direction:column;gap:7px;}
.ti{
  padding:11px 15px;border-radius:9px;font-size:12px;font-weight:600;
  min-width:260px;max-width:340px;display:flex;align-items:center;gap:9px;
  animation:slideR .28s ease;box-shadow:0 8px 24px rgba(0,0,0,.45);
}
.ti-s{background:#0d2818;border:1px solid rgba(34,197,94,.35);color:var(--green);}
.ti-e{background:#1f0a0a;border:1px solid rgba(239,68,68,.35);color:var(--red);}
.ti-i{background:#0a1520;border:1px solid rgba(0,229,255,.25);color:var(--accent);}
@keyframes slideR{from{opacity:0;transform:translateX(18px)}to{opacity:1;transform:translateX(0)}}

/* OWNER SELECT */
.owner-info{margin-top:6px;font-size:12px;color:var(--text3);}
.owner-info.ok{color:var(--green);}

/* REFRESH BTN */
.rbtn{
  width:30px;height:30px;border-radius:7px;background:var(--surface2);
  border:1px solid var(--border);cursor:pointer;display:flex;align-items:center;
  justify-content:center;color:var(--text2);font-size:13px;transition:all .18s;margin-left:auto;
}
.rbtn:hover{border-color:var(--accent);color:var(--accent);}

@media(max-width:768px){
  .stats-grid{grid-template-columns:1fr 1fr;}
  .fg2{grid-template-columns:1fr;}
  .fg2.c3{grid-template-columns:1fr;}
  .chart-grid{grid-template-columns:1fr;}
  #main{padding:14px;padding-top:58px;}
}
</style>
</head>
<body>

<button id="tog" onclick="toggleSB()" title="Toggle Sidebar">☰</button>

<aside id="sidebar">
  <div class="sb-head">
    <div class="sb-logo">R</div>
    <div>
      <div class="sb-title">RenzyDev Panel</div>
      <div class="sb-sub">v2.0 · Manager</div>
    </div>
  </div>
  <nav class="sb-nav">
    <div class="sb-sec">Overview</div>
    <div class="sb-item active" onclick="nav('dashboard',this);loadDash()">
      <span class="ic">📊</span> Dashboard
    </div>
    <div class="sb-sec">Manajemen</div>
    <div class="sb-item" onclick="nav('create-account',this)">
      <span class="ic">👤</span> Buat Akun
    </div>
    <div class="sb-item" onclick="nav('create-server',this)">
      <span class="ic">🖥️</span> Buat Server
    </div>
    <div class="sb-sec">Data</div>
    <div class="sb-item" onclick="nav('list-users',this);loadUsers()">
      <span class="ic">👥</span> List User
    </div>
    <div class="sb-item" onclick="nav('list-servers',this);loadServers()">
      <span class="ic">📦</span> List Server
    </div>
    <div class="sb-item" onclick="nav('list-nests',this);loadNests()">
      <span class="ic">🥚</span> List Nest
    </div>
    <div class="sb-item" onclick="nav('history',this);loadHistory()">
      <span class="ic">🕓</span> Riwayat
    </div>
  </nav>
  <div class="sb-foot">
    <div class="sb-status">
      <div class="dot-live"></div>
      <span>Panel Terhubung</span>
    </div>
  </div>
</aside>

<main id="main">

<!-- DASHBOARD -->
<div id="page-dashboard" class="page active">
  <div class="ph">
    <div class="ph-title">📊 <span>Dashboard</span></div>
    <div class="ph-sub">Overview panel Pterodactyl</div>
  </div>
  <div id="dash-alert"></div>
  <div class="stats-grid" id="dash-stats">
    <div class="loading"><div class="spin"></div>Loading...</div>
  </div>
  <div class="chart-grid" id="dash-charts" style="display:none">
    <div class="chart-card">
      <div class="chart-title">📈 Distribusi User</div>
      <canvas id="chartUser" height="160"></canvas>
    </div>
    <div class="chart-card">
      <div class="chart-title">🖥️ Status Server</div>
      <canvas id="chartServer" height="160"></canvas>
    </div>
  </div>
  <div class="card" id="dash-nodes-card" style="display:none">
    <div class="card-h"><span>🌐</span><h3>Node Info</h3><button class="rbtn" onclick="loadDash()">↻</button></div>
    <div class="card-b" id="dash-nodes"></div>
  </div>
</div>

<!-- CREATE ACCOUNT -->
<div id="page-create-account" class="page">
  <div class="ph">
    <div class="ph-title">Buat <span>Akun</span></div>
    <div class="ph-sub">Tambah akun baru ke panel Pterodactyl</div>
  </div>
  <div id="alert-acc"></div>
  <div class="card">
    <div class="card-h"><span>👤</span><h3>Informasi Akun</h3></div>
    <div class="card-b">
      <div class="fg2">
        <div class="fg"><label>Email</label><input type="email" id="a-email" placeholder="contoh@gmail.com"/></div>
        <div class="fg"><label>Username</label><input type="text" id="a-user" placeholder="username"/></div>
        <div class="fg"><label>First Name</label><input type="text" id="a-fn" placeholder="Nama depan"/></div>
        <div class="fg"><label>Last Name</label><input type="text" id="a-ln" placeholder="Nama belakang"/></div>
        <div class="fg"><label>Password</label><input type="password" id="a-pass" placeholder="Password kuat"/></div>
        <div class="fg">
          <label>Role</label>
          <select id="a-role"><option value="member">Member</option><option value="administrator">Administrator</option></select>
        </div>
        <div class="fg"><label>Default Language</label><input value="English" disabled style="opacity:.4"/></div>
      </div>
      <div class="fa">
        <button class="btn btn-ghost" onclick="resetAcc()">Reset</button>
        <button class="btn btn-primary" id="btn-acc" onclick="createAccount()">✨ Buat Akun</button>
      </div>
    </div>
  </div>
</div>

<!-- CREATE SERVER -->
<div id="page-create-server" class="page">
  <div class="ph">
    <div class="ph-title">Buat <span>Server</span></div>
    <div class="ph-sub">Provisioning server — pilih Nest & Egg sesuai kebutuhan</div>
  </div>
  <div id="alert-srv"></div>

  <!-- Phone -->
  <div class="phone-box">
    <h4>📱 Nomor WhatsApp Buyer</h4>
    <div class="fg">
      <label>Nomor HP</label>
      <input type="text" id="s-phone" placeholder="628xxx atau 08xxx" oninput="validatePhone()"/>
      <div class="hint" id="phone-hint"></div>
    </div>
  </div>

  <!-- Core -->
  <div class="card">
    <div class="card-h"><span>📋</span><h3>Core Details</h3></div>
    <div class="card-b">
      <div class="fg2">
        <div class="fg"><label>Server Name</label><input type="text" id="s-name" placeholder="Nama server"/></div>
        <div class="fg">
          <label>Server Owner</label>
          <div style="display:flex;gap:7px">
            <select id="s-owner" onchange="onOwnerSel()" style="flex:1">
              <option value="">⏳ Loading users...</option>
            </select>
            <button class="btn btn-ghost" style="padding:9px 11px;flex-shrink:0" onclick="loadOwners()" title="Refresh">↻</button>
          </div>
          <div class="owner-info" id="owner-info"></div>
        </div>
        <div class="fg"><label>Password (untuk notif WA)</label><input type="password" id="s-opass" placeholder="Password owner"/></div>
        <div class="fg full"><label>Deskripsi</label><textarea id="s-desc" placeholder="Deskripsi server..."></textarea></div>
      </div>
    </div>
  </div>

  <!-- Node + Nest + Egg -->
  <div class="card">
    <div class="card-h"><span>🌐</span><h3>Allocation & Egg Config</h3></div>
    <div class="card-b">
      <div class="fg2 c3">
        <div class="fg">
          <label>Node</label>
          <select id="s-node"><option value="">Loading nodes...</option></select>
        </div>
        <div class="fg">
          <label>Nest</label>
          <div style="display:flex;gap:7px">
            <select id="s-nest" onchange="onNestChange()" style="flex:1">
              <option value="">⏳ Loading nest...</option>
            </select>
            <button class="btn btn-ghost" style="padding:9px 11px;flex-shrink:0" onclick="loadNestSelect()" title="Refresh">↻</button>
          </div>
        </div>
        <div class="fg">
          <label>Egg</label>
          <div style="display:flex;gap:7px">
            <select id="s-egg" onchange="onEggChange()" style="flex:1">
              <option value="">— Pilih Nest dulu —</option>
            </select>
            <button class="btn btn-ghost" style="padding:9px 11px;flex-shrink:0" onclick="reloadEggs()" title="Refresh eggs" id="btn-reload-eggs" disabled>↻</button>
          </div>
          <div class="hint" id="egg-hint"></div>
        </div>
      </div>
      <div style="margin-top:9px;font-size:11px;color:var(--text3)">📌 Default & Additional allocation otomatis · Egg menentukan docker image & startup command</div>
    </div>
  </div>

  <!-- Feature Limits -->
  <div class="card">
    <div class="card-h"><span>⚙️</span><h3>Feature Limits</h3></div>
    <div class="card-b">
      <div class="fg2 c3">
        <div class="fg"><label>Database Limit</label><select id="s-db"></select></div>
        <div class="fg"><label>Backup Limit</label><select id="s-bk"></select></div>
        <div class="fg"><label>Allocation Limit</label><select id="s-al"></select></div>
      </div>
    </div>
  </div>

  <!-- Resources -->
  <div class="card">
    <div class="card-h"><span>💾</span><h3>Resource Management</h3></div>
    <div class="card-b">
      <div class="fg2 c3">
        <div class="fg"><label>CPU Limit</label><select id="s-cpu"></select></div>
        <div class="fg"><label>Memory (RAM)</label><select id="s-mem"></select></div>
        <div class="fg"><label>Disk Space</label><select id="s-disk"></select></div>
        <div class="fg"><label>CPU Pinning</label><input value="Default (otomatis)" disabled style="opacity:.4"/></div>
        <div class="fg"><label>Swap</label><input value="Default (0)" disabled style="opacity:.4"/></div>
        <div class="fg"><label>Block IO Weight</label><input value="Default (500)" disabled style="opacity:.4"/></div>
      </div>

      <div class="sdiv">Preview Pesan WhatsApp</div>
      <div class="wa-preview-wrap">
        <div class="wa-preview" id="wa-prev">_Isi data owner dan password untuk preview pesan WA_</div>
        <button class="copy-btn" id="copy-wa-btn" onclick="copyWA()">📋 Copy</button>
      </div>
      <div style="margin-top:9px;display:flex;gap:8px;justify-content:flex-end">
        <button class="btn btn-info btn-sm" onclick="updateWAPreview()">🔄 Update Preview</button>
        <button class="btn btn-ghost btn-sm" onclick="editWAPreview()" id="edit-wa-btn">✏️ Edit Pesan</button>
      </div>

      <div class="fa">
        <button class="btn btn-ghost" onclick="resetSrv()">Reset</button>
        <button class="btn btn-primary" id="btn-srv" onclick="createServer()">🚀 Buat Server & Kirim WA</button>
      </div>
    </div>
  </div>
</div>

<!-- LIST USERS -->
<div id="page-list-users" class="page">
  <div class="ph"><div class="ph-title">List <span>User</span></div><div class="ph-sub">Semua pengguna terdaftar</div></div>
  <div class="card">
    <div class="card-h"><span>👥</span><h3>Daftar User</h3><button class="rbtn" onclick="loadUsers()">↻</button></div>
    <div class="card-b">
      <div class="search-bar"><input type="text" id="search-users" placeholder="Cari username atau email..." oninput="filterUsers()"/></div>
      <div id="users-content"><div class="loading"><div class="spin"></div>Loading...</div></div>
    </div>
  </div>
</div>

<!-- LIST SERVERS -->
<div id="page-list-servers" class="page">
  <div class="ph"><div class="ph-title">List <span>Server</span></div><div class="ph-sub">Semua server aktif</div></div>
  <div class="card">
    <div class="card-h"><span>📦</span><h3>Daftar Server</h3><button class="rbtn" onclick="loadServers()">↻</button></div>
    <div class="card-b">
      <div class="search-bar"><input type="text" id="search-servers" placeholder="Cari nama server..." oninput="filterServers()"/></div>
      <div id="servers-content"><div class="loading"><div class="spin"></div>Loading...</div></div>
    </div>
  </div>
</div>

<!-- LIST NESTS -->
<div id="page-list-nests" class="page">
  <div class="ph"><div class="ph-title">List <span>Nest</span></div><div class="ph-sub">Semua nest tersedia</div></div>
  <div class="card">
    <div class="card-h"><span>🥚</span><h3>Daftar Nest</h3><button class="rbtn" onclick="loadNests()">↻</button></div>
    <div class="card-b" id="nests-content"><div class="loading"><div class="spin"></div>Loading...</div></div>
  </div>
</div>

<!-- HISTORY -->
<div id="page-history" class="page">
  <div class="ph"><div class="ph-title">📋 <span>Riwayat</span></div><div class="ph-sub">Server yang pernah dibuat (sesi ini)</div></div>
  <div class="card">
    <div class="card-h"><span>🕓</span><h3>Riwayat Pembuatan Server</h3><button class="rbtn" onclick="loadHistory()">↻</button></div>
    <div class="card-b" id="history-content"><div class="loading"><div class="spin"></div>Loading...</div></div>
  </div>
</div>

</main>

<!-- MODAL: Detail Server -->
<div class="modal-overlay" id="modal-detail">
  <div class="modal">
    <div class="modal-h">
      <h3>🖥️ Detail Server</h3>
      <button class="modal-close" onclick="closeModal('modal-detail')">✕</button>
    </div>
    <div id="modal-detail-body"></div>
  </div>
</div>

<!-- MODAL: Edit Server -->
<div class="modal-overlay" id="modal-edit">
  <div class="modal">
    <div class="modal-h">
      <h3>✏️ Edit Server</h3>
      <button class="modal-close" onclick="closeModal('modal-edit')">✕</button>
    </div>
    <div id="modal-edit-body">
      <input type="hidden" id="edit-srv-id"/>
      <div class="fg2">
        <div class="fg"><label>CPU Limit</label><select id="edit-cpu"></select></div>
        <div class="fg"><label>Memory (RAM)</label><select id="edit-mem"></select></div>
        <div class="fg"><label>Disk Space</label><select id="edit-disk"></select></div>
        <div class="fg"><label>Database Limit</label><select id="edit-db"></select></div>
        <div class="fg"><label>Backup Limit</label><select id="edit-bk"></select></div>
        <div class="fg"><label>Allocation Limit</label><select id="edit-al"></select></div>
      </div>
      <div class="fa">
        <button class="btn btn-ghost" onclick="closeModal('modal-edit')">Batal</button>
        <button class="btn btn-primary" id="btn-edit-srv" onclick="submitEditServer()">💾 Simpan</button>
      </div>
    </div>
  </div>
</div>

<!-- TOAST -->
<div id="toast"></div>

<script>
// ===================== INIT SELECTS =====================
function buildOpts(sel, items){
  sel.innerHTML = items.map(i=>'<option value="'+i.v+'">'+i.l+'</option>').join('');
}
function initSelects(){
  // 1-10 selects
  const t10 = Array.from({length:10},(_,i)=>({v:i+1,l:i+1}));
  ['s-db','s-bk','s-al','edit-db','edit-bk','edit-al'].forEach(id=>{
    const el = document.getElementById(id);
    if(el) buildOpts(el, t10);
  });
  // CPU 100-500
  const cpuOpts = [100,200,300,400,500].map(v=>({v,l:v+'% ('+(v/100)+' Core)'}));
  ['s-cpu','edit-cpu'].forEach(id=>{
    const el = document.getElementById(id);
    if(el) buildOpts(el, cpuOpts);
  });
  // RAM 1-50 GB
  const gbOpts = Array.from({length:50},(_,i)=>({v:i+1,l:(i+1)+' GB ('+(((i+1)*1024))+' MB)'}));
  ['s-mem','edit-mem','s-disk','edit-disk'].forEach(id=>{
    const el = document.getElementById(id);
    if(el) buildOpts(el, gbOpts);
  });
}
initSelects();

// ===================== SIDEBAR =====================
let sbOpen = true;
function toggleSB(){
  sbOpen = !sbOpen;
  document.getElementById('sidebar').classList.toggle('closed',!sbOpen);
  document.getElementById('main').classList.toggle('exp',!sbOpen);
}
function nav(page, el){
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('.sb-item').forEach(n=>n.classList.remove('active'));
  document.getElementById('page-'+page).classList.add('active');
  if(el) el.classList.add('active');
}

// ===================== TOAST =====================
function toast(msg, type='s'){
  const el = document.createElement('div');
  el.className = 'ti ti-'+type;
  el.innerHTML = (type==='s'?'✅':type==='e'?'❌':'ℹ️')+' '+msg;
  document.getElementById('toast').appendChild(el);
  setTimeout(()=>el.style.opacity='0',3200);
  setTimeout(()=>el.remove(),3600);
}
function alert$(id, msg, type='s'){
  const icons = {s:'✅',e:'❌',i:'ℹ️'};
  document.getElementById(id).innerHTML = msg ?
    '<div class="alert alert-'+type+'"><span>'+icons[type]+'</span><span>'+msg+'</span></div>' : '';
  if(msg) setTimeout(()=>document.getElementById(id).innerHTML='', 7000);
}

// ===================== MODAL =====================
function openModal(id){ document.getElementById(id).classList.add('open'); }
function closeModal(id){ document.getElementById(id).classList.remove('open'); }

// ===================== DASHBOARD =====================
let dashChart1=null, dashChart2=null;
async function loadDash(){
  document.getElementById('dash-stats').innerHTML = '<div class="loading"><div class="spin"></div>Loading...</div>';
  document.getElementById('dash-charts').style.display='none';
  document.getElementById('dash-nodes-card').style.display='none';
  try {
    const res = await fetch('/api/dashboard');
    const d = await res.json();
    if(!d.success){ alert$('dash-alert','Gagal load dashboard: '+d.message,'e'); return; }
    const x = d.data;
    const stats = [
      {icon:'👥',val:x.total_users,label:'Total User',color:'#7c3aed'},
      {icon:'📦',val:x.total_servers,label:'Total Server',color:'#06b6d4'},
      {icon:'🌐',val:x.total_nodes,label:'Total Node',color:'#22c55e'},
      {icon:'🔴',val:x.suspended,label:'Server Suspend',color:'#ef4444'},
    ];
    document.getElementById('dash-stats').innerHTML = stats.map(s=>
      '<div class="stat-card" style="--stat-color:'+s.color+'"><div class="stat-icon">'+s.icon+'</div><div class="stat-val">'+s.val+'</div><div class="stat-label">'+s.label+'</div></div>'
    ).join('');

    // Charts
    document.getElementById('dash-charts').style.display='grid';
    if(dashChart1){ dashChart1.destroy(); }
    if(dashChart2){ dashChart2.destroy(); }
    const adminCount = x.admin_count || 0;
    const memberCount = (x.total_users||0) - adminCount;
    dashChart1 = new Chart(document.getElementById('chartUser'),{
      type:'doughnut',
      data:{labels:['Admin','Member'],datasets:[{data:[adminCount,memberCount],backgroundColor:['#7c3aed','#06b6d4'],borderWidth:0}]},
      options:{plugins:{legend:{labels:{color:'#8b99ae',font:{family:'Syne',size:12}}}},cutout:'68%'}
    });
    const activeCount = (x.total_servers||0) - (x.suspended||0);
    dashChart2 = new Chart(document.getElementById('chartServer'),{
      type:'doughnut',
      data:{labels:['Aktif','Suspend'],datasets:[{data:[activeCount,x.suspended||0],backgroundColor:['#22c55e','#ef4444'],borderWidth:0}]},
      options:{plugins:{legend:{labels:{color:'#8b99ae',font:{family:'Syne',size:12}}}},cutout:'68%'}
    });

    // Nodes
    if(x.nodes && x.nodes.length > 0){
      document.getElementById('dash-nodes-card').style.display='';
      document.getElementById('dash-nodes').innerHTML = '<div class="tw"><table><thead><tr><th>ID</th><th>Nama Node</th><th>RAM (MB)</th><th>Disk (MB)</th></tr></thead><tbody>'+
        x.nodes.map(n=>'<tr><td><strong>'+n.attributes.id+'</strong></td><td>'+n.attributes.name+'</td><td>'+n.attributes.memory+'</td><td>'+n.attributes.disk+'</td></tr>').join('')+
      '</tbody></table></div>';
    }
    if(x.errors && x.errors.length>0) alert$('dash-alert','Beberapa data gagal dimuat: '+x.errors.join(', '),'i');
  } catch(e){ alert$('dash-alert','Error: '+e.message,'e'); }
}

// ===================== PHONE VALIDATE =====================
let phoneValidated = false;
async function validatePhone(){
  const phone = document.getElementById('s-phone').value.trim();
  const hint = document.getElementById('phone-hint');
  const inp = document.getElementById('s-phone');
  if(!phone){ hint.textContent=''; inp.className=''; phoneValidated=false; return; }
  try {
    const res = await fetch('/api/validate-phone',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({phone})});
    const d = await res.json();
    if(d.success){
      hint.textContent='✅ Valid → '+d.data; hint.className='hint ok';
      inp.className='valid'; phoneValidated=true;
    } else {
      hint.textContent='❌ '+d.message; hint.className='hint err';
      inp.className='invalid'; phoneValidated=false;
    }
  } catch(e){ hint.textContent=''; phoneValidated=false; }
}

// ===================== WA PREVIEW =====================
let waEdited = false;
function updateWAPreview(){
  const sel = document.getElementById('s-owner');
  const pass = document.getElementById('s-opass').value;
  let email='email@gmail.com', username='username';
  if(sel.value){ try{ const d=JSON.parse(sel.value); email=d.email; username=d.username; }catch(e){} }
  const msg = buildWAMsg(email,username,pass||'••••••••');
  if(!waEdited) document.getElementById('wa-prev').textContent = msg;
}
function buildWAMsg(email,user,pass){
  return '________📦KOTAK PESANAN ANDA________\n_selamat pesanan anda sudah terkonfirmasi oleh owner_\n\n_data data account anda_\n_gmail : '+email+'_\n_user : '+user+'_\n_password : '+pass+'_\n\n_link untuk masuk ke hosting_\n_link panel : {{.PanelLink}}_\n_link phpmyadmin : {{.PhpMyAdminLink}}_\n\n*________⚠️RULES / TOS________*\n_1.dilarang menggunakan script bertujuan ddos/hacking/bypass_\n_2.dilarang mencoba otak Atik sistem operasi_\n_3.jika account hilang/dicuri teman tidak ada refund_\n_4.refund aktif selama 7 hari_';
}
function editWAPreview(){
  const prev = document.getElementById('wa-prev');
  const btn = document.getElementById('edit-wa-btn');
  if(!waEdited){
    prev.contentEditable='true'; prev.style.border='1px solid var(--accent)';
    btn.textContent='✅ Selesai Edit'; waEdited=true; prev.focus();
  } else {
    prev.contentEditable='false'; prev.style.border='';
    btn.textContent='✏️ Edit Pesan'; waEdited=false;
  }
}
function copyWA(){
  const text = document.getElementById('wa-prev').textContent;
  navigator.clipboard.writeText(text).then(()=>{
    const btn = document.getElementById('copy-wa-btn');
    btn.textContent='✅ Copied!'; btn.classList.add('copied');
    setTimeout(()=>{ btn.textContent='📋 Copy'; btn.classList.remove('copied'); },2000);
  }).catch(()=>toast('Gagal copy','e'));
}

// ===================== OWNER SELECT =====================
async function loadOwners(){
  const sel = document.getElementById('s-owner');
  sel.innerHTML='<option value="">⏳ Loading...</option>';
  document.getElementById('owner-info').textContent='';
  try {
    const res = await fetch('/api/list-users');
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    sel.innerHTML='<option value="">-- Pilih Owner --</option>';
    (d.data||[]).forEach(u=>{
      const a=u.attributes;
      const opt=document.createElement('option');
      opt.value=JSON.stringify({email:a.email,username:a.username});
      opt.textContent=a.username+' ('+a.email+')';
      sel.appendChild(opt);
    });
  } catch(e){ sel.innerHTML='<option value="">❌ Gagal - '+e.message+'</option>'; }
}
function onOwnerSel(){
  const sel = document.getElementById('s-owner');
  const info = document.getElementById('owner-info');
  if(!sel.value){ info.textContent=''; info.className='owner-info'; return; }
  try {
    const d=JSON.parse(sel.value);
    info.innerHTML='✅ <strong>'+d.username+'</strong> · '+d.email;
    info.className='owner-info ok';
  } catch(e){}
  updateWAPreview();
}

// ===================== LOAD NODES =====================
async function loadNodes(){
  try {
    const res = await fetch('/api/nodes');
    const d = await res.json();
    const sel = document.getElementById('s-node');
    sel.innerHTML='';
    if(d.data && d.data.length>0){
      d.data.forEach(n=>{
        const opt=document.createElement('option');
        opt.value=n.attributes.id;
        opt.textContent=n.attributes.name+' (ID: '+n.attributes.id+')';
        sel.appendChild(opt);
      });
    } else { sel.innerHTML='<option value="">Tidak ada node</option>'; }
  } catch(e){ document.getElementById('s-node').innerHTML='<option value="">Gagal load</option>'; }
}

// ===================== NEST & EGG =====================
async function loadNestSelect(){
  const sel = document.getElementById('s-nest');
  sel.innerHTML='<option value="">⏳ Loading...</option>';
  document.getElementById('s-egg').innerHTML='<option value="">— Pilih Nest dulu —</option>';
  document.getElementById('btn-reload-eggs').disabled=true;
  document.getElementById('egg-hint').textContent='';
  try {
    const res = await fetch('/api/list-nests');
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    const nests = d.data||[];
    sel.innerHTML='<option value="">— Pilih Nest —</option>';
    nests.forEach(n=>{
      const opt=document.createElement('option');
      opt.value=n.attributes.id;
      opt.textContent=n.attributes.name+' (ID: '+n.attributes.id+')';
      sel.appendChild(opt);
    });
    // Auto select jika hanya 1 nest
    if(nests.length===1){
      sel.value=nests[0].attributes.id;
      onNestChange();
    }
  } catch(e){
    sel.innerHTML='<option value="">❌ Gagal load: '+e.message+'</option>';
    toast('Gagal load nest: '+e.message,'e');
  }
}

async function onNestChange(){
  const nestId = document.getElementById('s-nest').value;
  const eggSel = document.getElementById('s-egg');
  const hint = document.getElementById('egg-hint');
  const reloadBtn = document.getElementById('btn-reload-eggs');

  eggSel.innerHTML='<option value="">⏳ Loading eggs...</option>';
  hint.textContent=''; reloadBtn.disabled=true;

  if(!nestId){
    eggSel.innerHTML='<option value="">— Pilih Nest dulu —</option>';
    return;
  }
  try {
    const res = await fetch('/api/eggs?nest_id='+nestId);
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    const eggs = d.data||[];
    eggSel.innerHTML='<option value="">— Pilih Egg —</option>';
    eggs.forEach(e=>{
      const opt=document.createElement('option');
      opt.value=e.attributes.id;
      opt.textContent=e.attributes.name+' (ID: '+e.attributes.id+')';
      opt.dataset.docker=e.attributes.docker_image||'';
      eggSel.appendChild(opt);
    });
    reloadBtn.disabled=false;
    // Auto select jika hanya 1 egg
    if(eggs.length===1){
      eggSel.value=eggs[0].attributes.id;
      onEggChange();
    }
    hint.textContent='✅ '+eggs.length+' egg tersedia';
    hint.className='hint ok';
  } catch(e){
    eggSel.innerHTML='<option value="">❌ Gagal load</option>';
    hint.textContent='❌ '+e.message; hint.className='hint err';
    toast('Gagal load eggs: '+e.message,'e');
  }
}

function reloadEggs(){ onNestChange(); }

function onEggChange(){
  const eggSel = document.getElementById('s-egg');
  const opt = eggSel.options[eggSel.selectedIndex];
  const hint = document.getElementById('egg-hint');
  if(!opt||!opt.value){ hint.textContent=''; return; }
  const docker = opt.dataset.docker||'-';
  hint.textContent='🐳 '+docker;
  hint.className='hint';
  hint.style.color='var(--text3)';
}

// ===================== RESET =====================
function resetAcc(){
  ['a-email','a-user','a-fn','a-ln','a-pass'].forEach(id=>document.getElementById(id).value='');
  document.getElementById('a-role').value='member';
}
function resetSrv(){
  ['s-name','s-phone','s-opass','s-desc'].forEach(id=>document.getElementById(id).value='');
  document.getElementById('phone-hint').textContent='';
  document.getElementById('s-phone').className='';
  document.getElementById('owner-info').textContent='';
  document.getElementById('egg-hint').textContent='';
  document.getElementById('s-egg').innerHTML='<option value="">— Pilih Nest dulu —</option>';
  document.getElementById('s-nest').value='';
  document.getElementById('btn-reload-eggs').disabled=true;
  document.getElementById('wa-prev').textContent='_Isi data owner dan password untuk preview pesan WA_';
  waEdited=false; phoneValidated=false;
}

// ===================== CREATE ACCOUNT =====================
async function createAccount(){
  const btn = document.getElementById('btn-acc');
  const email=document.getElementById('a-email').value.trim();
  const user=document.getElementById('a-user').value.trim();
  const fn=document.getElementById('a-fn').value.trim();
  const ln=document.getElementById('a-ln').value.trim();
  const pass=document.getElementById('a-pass').value;
  const role=document.getElementById('a-role').value;
  if(!email||!user||!pass){ alert$('alert-acc','Email, username, dan password wajib diisi!','e'); return; }
  btn.disabled=true; btn.textContent='⏳ Membuat...';
  try {
    const res = await fetch('/api/create-account',{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({email,username:user,first_name:fn,last_name:ln,password:pass,role})});
    const d = await res.json();
    if(d.success){ alert$('alert-acc','Akun <strong>'+user+'</strong> berhasil dibuat!','s'); toast('Akun '+user+' berhasil dibuat!'); resetAcc(); }
    else { alert$('alert-acc',d.message,'e'); toast(d.message,'e'); }
  } catch(e){ alert$('alert-acc','Error: '+e.message,'e'); }
  btn.disabled=false; btn.innerHTML='✨ Buat Akun';
}

// ===================== CREATE SERVER =====================
async function createServer(){
  const btn = document.getElementById('btn-srv');
  const phone = document.getElementById('s-phone').value.trim();
  const name = document.getElementById('s-name').value.trim();
  const ownerRaw = document.getElementById('s-owner').value;
  const opass = document.getElementById('s-opass').value;
  const desc = document.getElementById('s-desc').value.trim();
  const nodeId = parseInt(document.getElementById('s-node').value);
  const nestId = parseInt(document.getElementById('s-nest').value);
  const eggId = parseInt(document.getElementById('s-egg').value);

  if(!phone){ alert$('alert-srv','Nomor WhatsApp buyer wajib diisi!','e'); return; }
  if(!name){ alert$('alert-srv','Nama server wajib diisi!','e'); return; }
  if(!ownerRaw){ alert$('alert-srv','Pilih owner server dari dropdown!','e'); return; }
  if(!nodeId){ alert$('alert-srv','Pilih Node terlebih dahulu!','e'); return; }
  if(!nestId){ alert$('alert-srv','Pilih Nest terlebih dahulu!','e'); return; }
  if(!eggId){ alert$('alert-srv','Pilih Egg terlebih dahulu!','e'); return; }

  let ownerEmail='', ownerUsername='';
  try{ const d=JSON.parse(ownerRaw); ownerEmail=d.email; ownerUsername=d.username; }
  catch(e){ alert$('alert-srv','Data owner tidak valid','e'); return; }

  // Pakai custom WA message jika sudah diedit
  const customMsg = waEdited ? document.getElementById('wa-prev').textContent : '';

  btn.disabled=true; btn.textContent='⏳ Membuat server...';

  const payload = {
    server_name:name, owner_email:ownerEmail, owner_username:ownerUsername,
    description:desc, node_id:nodeId,
    nest_id:nestId, egg_id:eggId,
    database_limit:parseInt(document.getElementById('s-db').value),
    backup_limit:parseInt(document.getElementById('s-bk').value),
    allocation_limit:parseInt(document.getElementById('s-al').value),
    cpu_limit:parseInt(document.getElementById('s-cpu').value),
    memory_gb:parseInt(document.getElementById('s-mem').value),
    disk_gb:parseInt(document.getElementById('s-disk').value),
    phone_number:phone, owner_password:opass,
  };

  try {
    const res = await fetch('/api/create-server',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
    const d = await res.json();
    if(d.success){
      alert$('alert-srv',d.message,'s'); toast('Server berhasil dibuat!');
      // Kirim WA custom jika sudah diedit
      if(customMsg){
        await fetch('/api/send-wa',{method:'POST',headers:{'Content-Type':'application/json'},
          body:JSON.stringify({phone:phone,message:customMsg})});
      }
      resetSrv();
    } else { alert$('alert-srv',d.message,'e'); toast(d.message,'e'); }
  } catch(e){ alert$('alert-srv','Error: '+e.message,'e'); }

  btn.disabled=false; btn.innerHTML='🚀 Buat Server & Kirim WA';
}

// ===================== USERS =====================
let allUsers = [];
async function loadUsers(){
  document.getElementById('users-content').innerHTML='<div class="loading"><div class="spin"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-users');
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    allUsers = d.data||[];
    renderUsers(allUsers);
  } catch(e){ document.getElementById('users-content').innerHTML='<div class="alert alert-e">❌ '+e.message+'</div>'; }
}
function filterUsers(){
  const q = document.getElementById('search-users').value.toLowerCase();
  renderUsers(allUsers.filter(u=>u.attributes.username.toLowerCase().includes(q)||u.attributes.email.toLowerCase().includes(q)));
}
function renderUsers(users){
  if(!users.length){ document.getElementById('users-content').innerHTML='<div class="empty"><div class="ei">👥</div><p>Belum ada user</p></div>'; return; }
  document.getElementById('users-content').innerHTML='<div class="tw"><table><thead><tr><th>ID</th><th>Username</th><th>Email</th><th>Role</th><th>Aksi</th></tr></thead><tbody>'+
    users.map(u=>{
      const a=u.attributes;
      return '<tr><td>'+a.id+'</td><td><strong>'+a.username+'</strong></td><td>'+a.email+'</td>'+
        '<td><span class="badge '+(a.root_admin?'b-admin':'b-member')+'">'+(a.root_admin?'Admin':'Member')+'</span></td>'+
        '<td><div class="act-btns">'+
          '<button class="btn btn-danger btn-sm" onclick="deleteUser('+a.id+',\''+a.username+'\')">🗑️ Hapus</button>'+
        '</div></td></tr>';
    }).join('')+
  '</tbody></table></div>';
}
async function deleteUser(id, name){
  if(!confirm('Hapus user "'+name+'" (ID: '+id+')? Aksi ini tidak bisa dibatalkan!')) return;
  try {
    const res = await fetch('/api/delete-user',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({user_id:id})});
    const d = await res.json();
    if(d.success){ toast('User '+name+' dihapus'); loadUsers(); }
    else toast(d.message,'e');
  } catch(e){ toast('Error: '+e.message,'e'); }
}

// ===================== SERVERS =====================
let allServers = [];
async function loadServers(){
  document.getElementById('servers-content').innerHTML='<div class="loading"><div class="spin"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-servers');
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    allServers = d.data||[];
    renderServers(allServers);
  } catch(e){ document.getElementById('servers-content').innerHTML='<div class="alert alert-e">❌ '+e.message+'</div>'; }
}
function filterServers(){
  const q = document.getElementById('search-servers').value.toLowerCase();
  renderServers(allServers.filter(s=>s.attributes.name.toLowerCase().includes(q)));
}
function renderServers(servers){
  if(!servers.length){ document.getElementById('servers-content').innerHTML='<div class="empty"><div class="ei">📦</div><p>Belum ada server</p></div>'; return; }
  document.getElementById('servers-content').innerHTML='<div class="tw"><table><thead><tr><th>ID</th><th>Nama</th><th>CPU</th><th>RAM</th><th>Disk</th><th>Status</th><th>Aksi</th></tr></thead><tbody>'+
    servers.map(s=>{
      const a=s.attributes;
      const ram = (a.limits.memory/1024).toFixed(1)+'GB';
      const disk = (a.limits.disk/1024).toFixed(1)+'GB';
      const isSusp = a.suspended;
      return '<tr>'+
        '<td>'+a.id+'</td>'+
        '<td><strong>'+a.name+'</strong><br><span style="font-size:10px;color:var(--text3);font-family:\'JetBrains Mono\',monospace">'+a.uuid.substring(0,12)+'...</span></td>'+
        '<td>'+a.limits.cpu+'%</td>'+
        '<td>'+ram+'</td>'+
        '<td>'+disk+'</td>'+
        '<td><span class="badge '+(isSusp?'b-suspend':'b-active')+'">'+(isSusp?'Suspended':'Aktif')+'</span></td>'+
        '<td><div class="act-btns">'+
          '<button class="btn btn-info btn-sm" onclick="showDetail('+a.id+')">🔍</button>'+
          '<button class="btn btn-ghost btn-sm" onclick="openEdit('+a.id+','+a.limits.cpu+','+(a.limits.memory/1024)+','+(a.limits.disk/1024)+','+a.feature_limits.databases+','+a.feature_limits.backups+','+a.feature_limits.allocations+')">✏️</button>'+
          '<button class="btn btn-warn btn-sm" onclick="toggleSuspend('+a.id+','+isSusp+')">'+(isSusp?'▶️':'⏸️')+'</button>'+
          '<button class="btn btn-ghost btn-sm" onclick="reinstall('+a.id+',\''+a.name+'\')">🔄</button>'+
          '<button class="btn btn-danger btn-sm" onclick="deleteServer('+a.id+',\''+a.name+'\')">🗑️</button>'+
        '</div></td></tr>';
    }).join('')+
  '</tbody></table></div>';
}
async function showDetail(id){
  document.getElementById('modal-detail-body').innerHTML='<div class="loading"><div class="spin"></div>Loading...</div>';
  openModal('modal-detail');
  try {
    const res = await fetch('/api/server-detail?id='+id);
    const d = await res.json();
    if(!d.success){ document.getElementById('modal-detail-body').innerHTML='<div class="alert alert-e">❌ '+d.message+'</div>'; return; }
    const a = d.data.attributes;
    document.getElementById('modal-detail-body').innerHTML=
      '<div class="fg2">'+
      '<div class="fg"><label>Nama</label><input value="'+a.name+'" disabled/></div>'+
      '<div class="fg"><label>UUID</label><input value="'+a.uuid+'" style="font-family:\'JetBrains Mono\';font-size:11px" disabled/></div>'+
      '<div class="fg"><label>CPU</label><input value="'+a.limits.cpu+'%" disabled/></div>'+
      '<div class="fg"><label>Memory</label><input value="'+(a.limits.memory/1024).toFixed(1)+' GB ('+a.limits.memory+' MB)" disabled/></div>'+
      '<div class="fg"><label>Disk</label><input value="'+(a.limits.disk/1024).toFixed(1)+' GB ('+a.limits.disk+' MB)" disabled/></div>'+
      '<div class="fg"><label>Swap</label><input value="'+a.limits.swap+' MB" disabled/></div>'+
      '<div class="fg"><label>Database Limit</label><input value="'+a.feature_limits.databases+'" disabled/></div>'+
      '<div class="fg"><label>Backup Limit</label><input value="'+a.feature_limits.backups+'" disabled/></div>'+
      '<div class="fg"><label>Allocation Limit</label><input value="'+a.feature_limits.allocations+'" disabled/></div>'+
      '<div class="fg"><label>Status</label><input value="'+(a.suspended?'Suspended':'Aktif')+'" disabled style="color:'+(a.suspended?'var(--red)':'var(--green)')+'"/></div>'+
      '</div>';
  } catch(e){ document.getElementById('modal-detail-body').innerHTML='<div class="alert alert-e">❌ '+e.message+'</div>'; }
}
function openEdit(id,cpu,memGb,diskGb,db,bk,al){
  document.getElementById('edit-srv-id').value = id;
  const setVal = (selId, val) => {
    const el = document.getElementById(selId);
    if(el) el.value = Math.round(val);
  };
  setVal('edit-cpu',cpu);
  setVal('edit-mem',Math.round(memGb));
  setVal('edit-disk',Math.round(diskGb));
  setVal('edit-db',db);
  setVal('edit-bk',bk);
  setVal('edit-al',al);
  openModal('modal-edit');
}
async function submitEditServer(){
  const btn = document.getElementById('btn-edit-srv');
  const id = parseInt(document.getElementById('edit-srv-id').value);
  btn.disabled=true; btn.textContent='⏳ Menyimpan...';
  try {
    const res = await fetch('/api/edit-server',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({
      server_id:id,
      cpu_limit:parseInt(document.getElementById('edit-cpu').value),
      memory_gb:parseInt(document.getElementById('edit-mem').value),
      disk_gb:parseInt(document.getElementById('edit-disk').value),
      database_limit:parseInt(document.getElementById('edit-db').value),
      backup_limit:parseInt(document.getElementById('edit-bk').value),
      allocation_limit:parseInt(document.getElementById('edit-al').value),
    })});
    const d = await res.json();
    if(d.success){ toast('Server ID '+id+' berhasil diupdate'); closeModal('modal-edit'); loadServers(); }
    else toast(d.message,'e');
  } catch(e){ toast('Error: '+e.message,'e'); }
  btn.disabled=false; btn.innerHTML='💾 Simpan';
}
async function toggleSuspend(id, isSuspended){
  const action = isSuspended ? 'aktifkan' : 'suspend';
  if(!confirm('Yakin '+(isSuspended?'mengaktifkan':'mensuspend')+' server ID '+id+'?')) return;
  try {
    const res = await fetch('/api/suspend-server',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({server_id:id,suspended:!isSuspended})});
    const d = await res.json();
    if(d.success){ toast(d.message); loadServers(); }
    else toast(d.message,'e');
  } catch(e){ toast('Error: '+e.message,'e'); }
}
async function reinstall(id, name){
  if(!confirm('Reinstall server "'+name+'" (ID: '+id+')? Semua data server akan terhapus!')) return;
  try {
    const res = await fetch('/api/reinstall-server',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({server_id:id})});
    const d = await res.json();
    if(d.success){ toast(d.message,'i'); loadServers(); }
    else toast(d.message,'e');
  } catch(e){ toast('Error: '+e.message,'e'); }
}
async function deleteServer(id, name){
  if(!confirm('Hapus server "'+name+'" (ID: '+id+')? Aksi ini tidak bisa dibatalkan!')) return;
  try {
    const res = await fetch('/api/delete-server',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({server_id:id})});
    const d = await res.json();
    if(d.success){ toast('Server '+name+' dihapus'); loadServers(); }
    else toast(d.message,'e');
  } catch(e){ toast('Error: '+e.message,'e'); }
}

// ===================== NESTS =====================
async function loadNests(){
  document.getElementById('nests-content').innerHTML='<div class="loading"><div class="spin"></div>Loading...</div>';
  try {
    const res = await fetch('/api/list-nests');
    const d = await res.json();
    if(!d.success) throw new Error(d.message);
    const nests = d.data||[];
    if(!nests.length){ document.getElementById('nests-content').innerHTML='<div class="empty"><div class="ei">🥚</div><p>Belum ada nest</p></div>'; return; }
    document.getElementById('nests-content').innerHTML='<div class="tw"><table><thead><tr><th>ID</th><th>Nama</th></tr></thead><tbody>'+
      nests.map(n=>'<tr><td><strong>'+n.attributes.id+'</strong></td><td>'+n.attributes.name+'</td></tr>').join('')+
    '</tbody></table></div>';
  } catch(e){ document.getElementById('nests-content').innerHTML='<div class="alert alert-e">❌ '+e.message+'</div>'; }
}

// ===================== HISTORY =====================
async function loadHistory(){
  document.getElementById('history-content').innerHTML='<div class="loading"><div class="spin"></div>Loading...</div>';
  try {
    const res = await fetch('/api/history');
    const d = await res.json();
    const hist = d.data||[];
    if(!hist.length){ document.getElementById('history-content').innerHTML='<div class="empty"><div class="ei">🕓</div><p>Belum ada riwayat di sesi ini</p></div>'; return; }
    document.getElementById('history-content').innerHTML = hist.map(h=>
      '<div class="hist-item">'+
        '<div class="hist-num">#'+h.id+'</div>'+
        '<div class="hist-info">'+
          '<div class="hist-name">'+h.server_name+'</div>'+
          '<div class="hist-meta">👤 '+h.owner_username+' ('+h.owner_email+') · 📱 '+h.phone_number+' · CPU:'+h.cpu+'% RAM:'+h.memory_gb+'GB Disk:'+h.disk_gb+'GB</div>'+
          '<div class="hist-time">'+new Date(h.created_at).toLocaleString('id-ID')+'</div>'+
        '</div>'+
      '</div>'
    ).join('');
  } catch(e){ document.getElementById('history-content').innerHTML='<div class="alert alert-e">❌ '+e.message+'</div>'; }
}

// ===================== INIT =====================
loadDash();
loadNodes();
loadOwners();
loadNestSelect();
// Auto preview update on pass change
document.getElementById('s-opass').addEventListener('input', updateWAPreview);
</script>
</body>
</html>`

func main() {
	tmpl := template.Must(template.New("p").Parse(htmlPage))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" { http.NotFound(w, r); return }
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, struct {
			PanelLink, PhpMyAdminLink string
		}{PanelLink, PhpMyAdminLink})
	})

	http.HandleFunc("/api/create-account",  handleCreateAccount)
	http.HandleFunc("/api/create-server",   handleCreateServer)
	http.HandleFunc("/api/list-users",      handleListUsers)
	http.HandleFunc("/api/list-servers",    handleListServers)
	http.HandleFunc("/api/list-nests",      handleListNests)
	http.HandleFunc("/api/eggs",            handleListEggs)
	http.HandleFunc("/api/nodes",           handleGetNodes)
	http.HandleFunc("/api/delete-user",     handleDeleteUser)
	http.HandleFunc("/api/delete-server",   handleDeleteServer)
	http.HandleFunc("/api/suspend-server",  handleSuspendServer)
	http.HandleFunc("/api/reinstall-server",handleReinstallServer)
	http.HandleFunc("/api/edit-server",     handleEditServer)
	http.HandleFunc("/api/server-detail",   handleServerDetail)
	http.HandleFunc("/api/dashboard",       handleDashboard)
	http.HandleFunc("/api/history",         handleHistory)
	http.HandleFunc("/api/validate-phone",  handleValidatePhone)
	http.HandleFunc("/api/send-wa",         handleSendWA)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"ok","service":"RenzyDev Panel Manager v2"}`)
	})

	port := os.Getenv("PORT")
	if port == "" { port = Port }

	log.Printf("🚀 RenzyDev Panel v2.0 → :%s", port)
	if strings.Contains(PterodactylAPIKey, "MASUKKAN") { log.Println("⚠️  Set PterodactylAPIKey di config!") }
	if strings.Contains(FonnteAPIKey, "MASUKKAN") { log.Println("⚠️  Set FonnteAPIKey di config!") }

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
