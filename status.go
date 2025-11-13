package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// atomic write helper
func osWriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	TMP := path + ".tmp"
	if err := os.WriteFile(TMP, data, 0644); err != nil {
		return err
	}
	return os.Rename(TMP, path)
}

// exportStatusFile writes current service statuses to JSON atomically
func exportStatusFile(services []ServiceInfo, path string) error {
	status := getServicesStatus(services)
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return osWriteAtomic(path, data)
}

func loadStatusFile(path string) ([]Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var services []Service
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, err
	}
	return services, nil
}

func getServicesStatus(services []ServiceInfo) []Service {
	var result []Service
	for _, s := range services {
		var active, isSystemd bool
		unit := s.ServiceName
		if unit == "" {
			unit = s.SystemdName
		}
		if runtime.GOOS == "linux" && unit != "" {
			active = isSystemdServiceActive(unit)
			isSystemd = true
		} else if runtime.GOOS == "windows" && unit != "" {
			active = isWindowsServiceActive(unit) || isWindowsProcessActive(unit)
		} else if s.Port > 0 {
			active = isPortInUse(s.Port)
		}
		result = append(result, Service{Port: s.Port, Name: s.Name, Link: s.Link, Image: s.Image, ShowPort: s.ShowPort, SystemdName: unit, IsSystemd: isSystemd, Active: active, Controls: s.Controls, ControlsRun: s.ControlsRun, ControlsShut: s.ControlsShut})
	}
	return result
}

func isPortInUse(port int) bool {
	if port <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), envCfg.DialTimeout)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}
	cmd := exec.Command("ss", "-tuln")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), fmt.Sprintf(":%d ", port)) {
		return true
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), envCfg.DialTimeout)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

func isSystemdServiceActive(name string) bool {
	if runtime.GOOS != "linux" || name == "" {
		return false
	}
	return exec.Command("systemctl", "is-active", "--quiet", name).Run() == nil
}

// renderHTML builds page
func renderHTML(w http.ResponseWriter, services []Service, templatePath string) {
	var active, inactive []Service
	for _, s := range services {
		if s.Active {
			active = append(active, s)
		} else {
			inactive = append(inactive, s)
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Name < active[j].Name })
	sort.Slice(inactive, func(i, j int) bool { return inactive[i].Name < inactive[j].Name })
	services = append(active, inactive...)
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"getInitials": func(name string) string {
			for _, c := range name {
				if c != ' ' && c != '[' {
					return string(c)
				}
			}
			return "?"
		},
		"Year": func() int { return time.Now().Year() },
	}).ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.html", services); err != nil {
		http.Error(w, "template exec error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// track last exported state to detect status changes
var lastStatus = map[string]bool{} // key: name|port|systemd

func detectAndLogStatusChanges(prev map[string]bool, curr []Service) {
	now := time.Now()
	for _, s := range curr {
		key := fmt.Sprintf("%s|%d|%s", s.Name, s.Port, s.SystemdName)
		old, ok := prev[key]
		if !ok {
			prev[key] = s.Active
			continue
		}
		if old != s.Active {
			prev[key] = s.Active
			// pseudo ServiceInfo for logging
			si := ServiceInfo{Port: s.Port, Name: s.Name, ServiceName: s.SystemdName, SystemdName: s.SystemdName}
			state := "up"
			if !s.Active {
				state = "down"
			}
			_ = logAction(appCfg.LogFile, now, "monitor", "127.0.0.1", &si, "status", state)
		}
	}
}

// defaultServicesFromInfo builds a non-blocking placeholder status list
// (Active=false) to render UI instantly until live checks complete.
func defaultServicesFromInfo(infos []ServiceInfo) []Service {
	res := make([]Service, 0, len(infos))
	for _, s := range infos {
		unit := s.ServiceName
		if unit == "" {
			unit = s.SystemdName
		}
		isSystemd := runtime.GOOS == "linux" && unit != ""
		res = append(res, Service{
			Port:         s.Port,
			Name:         s.Name,
			Link:         s.Link,
			Image:        s.Image,
			ShowPort:     s.ShowPort,
			SystemdName:  unit,
			IsSystemd:    isSystemd,
			Active:       false,
			Controls:     s.Controls,
			ControlsRun:  s.ControlsRun,
			ControlsShut: s.ControlsShut,
		})
	}
	return res
}
