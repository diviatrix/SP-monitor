package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// statusExportInterval defines how often we refresh the status file (seconds) (optional env STATUS_INTERVAL)
var statusExportInterval = 5 * time.Second

var envCfg EnvConfig
var appCfg *Config

func absFromExe(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	exe, err := os.Executable()
	if err != nil {
		return p
	}
	return filepath.Join(filepath.Dir(exe), p)
}

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}
	appCfg = config

	// Apply defaults if fields are missing for backward compatibility
	svFile := config.ServicesFile
	if svFile == "" {
		svFile = "services.json"
	}
	webDir := config.WebDir
	if webDir == "" {
		webDir = "web"
	}
	templateFile := config.TemplateFile
	if templateFile == "" {
		templateFile = webDir + "/index.html"
	}
	if appCfg.LogFile == "" {
		appCfg.LogFile = "log.csv"
	}

	// Resolve absolute paths from executable directory (robust across OS)
	webDirAbs := absFromExe(webDir)
	templateFileAbs := absFromExe(templateFile)

	// Load environment settings via helper
	envCfg = LoadEnv()
	exportPathAbs := absFromExe(envCfg.ExportPath)
	importPathAbs := absFromExe(envCfg.ImportPath)
	statusFileWrite := filepath.Join(exportPathAbs, envCfg.ExportName)
	statusFileRead := filepath.Join(importPathAbs, envCfg.ImportName)
	statusExportInterval = envCfg.StatusInterval

	// Load services configuration
	servicesConfig, err := loadServicesConfig(svFile)
	if err != nil {
		log.Fatal("Error loading services config:", err)
	}

	// Background exporter with change detection (non-blocking startup)
	go func() {
		// initial snapshot + export
		curr := getServicesStatus(servicesConfig.Services)
		detectAndLogStatusChanges(lastStatus, curr)
		if err := exportStatusFile(servicesConfig.Services, statusFileWrite); err != nil {
			log.Println("Initial status export error:", err)
		}
		// periodic refresh
		ticker := time.NewTicker(statusExportInterval)
		defer ticker.Stop()
		for range ticker.C {
			curr := getServicesStatus(servicesConfig.Services)
			detectAndLogStatusChanges(lastStatus, curr)
			if err := exportStatusFile(servicesConfig.Services, statusFileWrite); err != nil {
				log.Println("Status export error:", err)
			}
		}
	}()

	// API routes
	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Login string `json:"login"`
			Pass  string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Login == appCfg.AdminLogin && body.Pass == appCfg.AdminPass && body.Login != "" {
			tok := newToken()
			sessions.Lock()
			sessions.tokens[tok] = body.Login
			sessions.Unlock()
			http.SetCookie(w, &http.Cookie{Name: "session", Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(24 * time.Hour)})
			respondJSON(w, map[string]any{"ok": true})
			return
		}
		respondJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false})
	})

	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if tok := getSessionToken(r); tok != "" {
			sessions.Lock()
			delete(sessions.tokens, tok)
			sessions.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", Expires: time.Unix(0, 0)})
		respondJSON(w, map[string]any{"ok": true})
	})

	http.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		user := authUser(r)
		respondJSON(w, map[string]any{"loggedIn": user != "", "user": user})
	})

	// Logs endpoint: return last N lines (without header)
	http.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		user := authUser(r)
		if user == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		limit := 200
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 2000 {
				limit = v
			}
		}
		lines := tailLogFile(appCfg.LogFile, limit)
		respondJSON(w, map[string]any{"lines": lines})
	})

	type actionRequest struct {
		Name        string `json:"name,omitempty"`
		ServiceName string `json:"service_name,omitempty"`
		SystemdName string `json:"systemd_name,omitempty"`
		Port        int    `json:"port,omitempty"`
	}

	actionHandler := func(kind string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			user := authUser(r)
			if user == "" {
				respondJSONCode(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			var req actionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondJSONCode(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
				return
			}
			// find target
			var target *ServiceInfo
			for i := range servicesConfig.Services {
				si := &servicesConfig.Services[i]
				if (req.Name != "" && eqFold(si.Name, req.Name)) ||
					(req.ServiceName != "" && eqFold(si.ServiceName, req.ServiceName)) ||
					(req.SystemdName != "" && eqFold(si.SystemdName, req.SystemdName)) ||
					(req.Port != 0 && si.Port == req.Port) {
					target = si
					break
				}
			}
			if target == nil {
				respondJSONCode(w, http.StatusNotFound, map[string]string{"error": "service not found"})
				return
			}
			// permissions
			if !target.Controls {
				respondJSONCode(w, http.StatusForbidden, map[string]string{"error": "controls disabled"})
				return
			}
			if kind == "start" && !target.ControlsRun {
				respondJSONCode(w, http.StatusForbidden, map[string]string{"error": "start disabled"})
				return
			}
			if kind == "stop" && !target.ControlsShut {
				respondJSONCode(w, http.StatusForbidden, map[string]string{"error": "stop disabled"})
				return
			}

			var err error
			if kind == "start" {
				err = startService(*target)
			} else {
				err = stopService(*target)
			}
			result := "ok"
			if err != nil {
				result = "error: " + err.Error()
				respondJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			} else {
				respondJSON(w, map[string]any{"ok": true})
				_ = exportStatusFile(servicesConfig.Services, statusFileWrite)
			}
			_ = logAction(appCfg.LogFile, time.Now(), user, clientIP(r), target, kind, result)
		}
	}

	http.Handle("/api/service/start", actionHandler("start"))
	http.Handle("/api/service/stop", actionHandler("stop"))

	// Expose exported status.json (optional consumption by clients)
	http.HandleFunc("/status.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, statusFileWrite)
	})

	// Page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		services, err := loadStatusFile(statusFileRead)
		if err != nil {
			// Do not block on live checks at first load; render quickly with defaults
			services = defaultServicesFromInfo(servicesConfig.Services)
		}
		renderHTML(w, services, templateFileAbs)
	})

	// Static
	http.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDirAbs, "styles.css"))
	})
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDirAbs, "favicon.ico"))
	})

	fmt.Printf("Server starting on port %d (export: %s, import: %s, interval: %s, os: %s)\n", config.Port, statusFileWrite, statusFileRead, statusExportInterval, runtime.GOOS)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

// small helper used only here
func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// tailLogFile returns last n lines excluding header
func tailLogFile(path string, n int) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(lines[0], "timestamp,") {
		startIdx = 1
	}
	body := lines[startIdx:]
	if len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
	}
	if n >= len(body) {
		return body
	}
	return body[len(body)-n:]
}
