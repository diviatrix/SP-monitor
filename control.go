package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func startService(s ServiceInfo) error {
	if s.RunPath != "" {
		cmd := exec.Command(s.RunPath)
		if dir := filepath.Dir(s.RunPath); dir != "" {
			cmd.Dir = dir
		}
		if len(s.RunEnv) > 0 {
			env := os.Environ()
			for k, v := range s.RunEnv {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			cmd.Env = env
		}
		return cmd.Start()
	}
	name := s.ServiceName
	if name == "" {
		name = s.SystemdName
	}
	if runtime.GOOS == "linux" && name != "" {
		return runCmd(exec.Command("systemctl", "start", name))
	}
	if runtime.GOOS == "windows" && name != "" {
		lname := strings.ToLower(name)
		if strings.HasSuffix(lname, ".exe") {
			return runCmd(exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Start-Process -FilePath '%s'", name)))
		}
		return runCmd(exec.Command("sc", "start", name))
	}
	return fmt.Errorf("unsupported start")
}

func stopService(s ServiceInfo) error {
	if s.RunPath != "" {
		base := filepath.Base(s.RunPath)
		if runtime.GOOS == "windows" {
			return runCmd(exec.Command("taskkill", "/IM", base, "/F"))
		}
		if err := runCmd(exec.Command("pkill", "-f", s.RunPath)); err == nil {
			return nil
		}
		return runCmd(exec.Command("killall", base))
	}
	name := s.ServiceName
	if name == "" {
		name = s.SystemdName
	}
	if runtime.GOOS == "linux" && name != "" {
		return runCmd(exec.Command("systemctl", "stop", name))
	}
	if runtime.GOOS == "windows" && name != "" {
		lname := strings.ToLower(name)
		if strings.HasSuffix(lname, ".exe") {
			return runCmd(exec.Command("taskkill", "/IM", name, "/F"))
		}
		return runCmd(exec.Command("sc", "stop", name))
	}
	return fmt.Errorf("unsupported stop")
}

func runCmd(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// prepend log line, enforce max size, no streaming
func logAction(logFile string, ts time.Time, user, ip string, s *ServiceInfo, action, result string) error {
	if logFile == "" {
		logFile = "log.csv"
	}
	header := "timestamp,user,ip,action,name,service_name,systemd_name,port,result\n"
	var existing []byte
	if b, err := os.ReadFile(logFile); err == nil {
		existing = b
	} else {
		existing = []byte(header)
	}
	if len(existing) == 0 || !strings.HasPrefix(string(existing), "timestamp,") {
		existing = []byte(header)
	}
	line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%d,%s\n", ts.Format(time.RFC3339), escCSV(user), escCSV(ip), escCSV(action), escCSV(s.Name), escCSV(s.ServiceName), escCSV(s.SystemdName), s.Port, escCSV(result))
	// split header from body
	bodyStart := strings.Index(string(existing), "\n")
	var head, body []byte
	if bodyStart >= 0 {
		head = existing[:bodyStart+1]
		body = existing[bodyStart+1:]
	} else {
		head = []byte(header)
		body = existing
	}
	content := append(append(head, []byte(line)...), body...)
	max := appCfg.LogMaxBytes
	if max > 0 && len(content) > max {
		// keep header then trim body
		if len(head) >= max {
			content = head[:max]
		} else {
			bodyMax := max - len(head)
			if bodyMax < len(body) {
				body = body[:bodyMax]
			}
			if j := strings.LastIndex(string(body), "\n"); j > 0 {
				body = body[:j+1]
			}
			content = append(head, body...)
		}
	}
	TMP := logFile + ".tmp"
	if err := os.WriteFile(TMP, content, 0644); err != nil {
		return err
	}
	return os.Rename(TMP, logFile)
}

func escCSV(s string) string {
	if strings.ContainsAny(s, ",\n\r\"") {
		return "\"" + strings.ReplaceAll(strings.ReplaceAll(s, "\"", "\"\""), "\n", " ") + "\""
	}
	return s
}
