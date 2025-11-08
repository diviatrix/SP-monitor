package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
)

// Config represents the configuration structure
type Config struct {
	Port     int    `json:"port"`
}

// ServicesConfig represents the services configuration
type ServicesConfig struct {
	Services []ServiceInfo `json:"services"`
}

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Port          int    `json:"port"`
	Name          string `json:"name"`
	Link          string `json:"link,omitempty"`
	Image         string `json:"image,omitempty"`
	ShowPort      bool   `json:"show_port,omitempty"`
	SystemdName   string `json:"systemd_name,omitempty"`
}

// Service represents a service with its status
type Service struct {
	Port        int
	Name        string
	Link        string
	Image       string
	ShowPort    bool
	SystemdName string
	IsSystemd   bool
	Active      bool
}

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Load services configuration
	servicesConfig, err := loadServicesConfig("services.json")
	if err != nil {
		log.Fatal("Error loading services config:", err)
	}

	// Set up HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get services information
		services := getServicesStatus(servicesConfig.Services)

		// Render HTML page
		renderHTML(w, services)
	})
	
	// Serve static files (CSS, JS, images)
	http.Handle("/styles.css", http.StripPrefix("/", http.FileServer(http.Dir("./web/"))))

	fmt.Printf("Server starting on port %d\n", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

// loadConfig loads the configuration from a JSON file
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// loadServicesConfig loads the services configuration from a JSON file
func loadServicesConfig(filename string) (*ServicesConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var servicesConfig ServicesConfig
	err = json.Unmarshal(data, &servicesConfig)
	if err != nil {
		return nil, err
	}

	return &servicesConfig, nil
}



// getServicesStatus checks the status of all configured services
func getServicesStatus(services []ServiceInfo) []Service {
	var result []Service

	for _, service := range services {
		var active bool
		var isSystemd bool
		
		// Check if it's a systemd service based on whether SystemdName is present
		if service.SystemdName != "" {
			active = isSystemdServiceActive(service.SystemdName)
			isSystemd = true
		} else {
			active = isPortInUse(service.Port)
			isSystemd = false
		}

		result = append(result, Service{
			Port:        service.Port,
			Name:        service.Name,
			Link:        service.Link,
			Image:       service.Image,
			ShowPort:    service.ShowPort,
			SystemdName: service.SystemdName,
			IsSystemd:   isSystemd,
			Active:      active,
		})
	}

	return result
}

// isPortInUse checks if a port is in use
func isPortInUse(port int) bool {
	cmd := exec.Command("ss", "-tuln")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), fmt.Sprintf(":%d ", port))
}

// isSystemdServiceActive checks if a systemd service is active
func isSystemdServiceActive(serviceName string) bool {
	if serviceName == "" {
		return false
	}
	
	cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
	err := cmd.Run()
	
	// If the command returns 0 (success), the service is active
	return err == nil
}

// renderHTML generates and serves the HTML page
func renderHTML(w http.ResponseWriter, services []Service) {
	// Separate active and inactive services
	var activeServices, inactiveServices []Service
	for _, service := range services {
		if service.Active {
			activeServices = append(activeServices, service)
		} else {
			inactiveServices = append(inactiveServices, service)
		}
	}

	// Sort active services alphabetically by name
	sort.Slice(activeServices, func(i, j int) bool {
		return activeServices[i].Name < activeServices[j].Name
	})

	// Sort inactive services alphabetically by name
	sort.Slice(inactiveServices, func(i, j int) bool {
		return inactiveServices[i].Name < inactiveServices[j].Name
	})

	// Combine active and inactive services
	services = append(activeServices, inactiveServices...)

	// Load the HTML template from the external file
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"getInitials": func(name string) string {
			if len(name) == 0 {
				return "?"
			}
			// Extract first letter of the first word
			for _, char := range name {
				if char != ' ' && char != '[' {
					return string(char)
				}
			}
			return "?"
		},
		"Year": func() int {
			return 2025 // Current year
		},
	}).ParseFiles("web/index.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "index.html", services)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
	}
}