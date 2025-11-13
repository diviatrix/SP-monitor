package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func isWindowsServiceActive(name string) bool {
	if name == "" {
		return false
	}
	out, err := exec.Command("sc", "query", name).Output()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "state") && strings.Contains(s, "running")
}

func isWindowsProcessActive(name string) bool {
	if name == "" {
		return false
	}
	lname := strings.ToLower(name)
	if strings.HasSuffix(lname, ".exe") {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name), "/FO", "CSV", "/NH").Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), fmt.Sprintf("\"%s\",", lname)) {
					return true
				}
			}
		}
	}
	out, err := exec.Command("tasklist", "/V", "/FO", "CSV", "/NH").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(strings.ToLower(line), lname) {
				return true
			}
		}
	}
	return isWindowsProcessActivePowerShell(name)
}

func isWindowsProcessActivePowerShell(name string) bool {
	if name == "" {
		return false
	}
	esc := strings.ReplaceAll(name, "'", "''")
	script := fmt.Sprintf("$n='%s'; $p = Get-Process | Where-Object { $_.MainWindowTitle -like \"*${n}*\" -or $_.Description -like \"*${n}*\" -or $_.Path -like \"*${n}*\" -or $_.ProcessName -like \"*${n}*\" }; if ($p) { exit 0 } else { exit 1 }", esc)
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run() == nil
}
