package main

// Config represents the configuration structure
// JSON-loaded from config.json
// Optional fields kept for backward compatibility
// and to avoid breaking existing deployments.
type Config struct {
	Port                int    `json:"port"`
	ServicesFile        string `json:"services_file,omitempty"`
	WebDir              string `json:"web_dir,omitempty"`
	TemplateFile        string `json:"template_file,omitempty"`
	AdminLogin          string `json:"admin_login,omitempty"`
	AdminPass           string `json:"admin_password,omitempty"`
	LogFile             string `json:"log_file,omitempty"`
	LogMaxBytes         int    `json:"log_max_bytes,omitempty"`
	CommonPasswordsFile string `json:"common_passwords_file,omitempty"`
}

// ServicesConfig represents the services configuration
// JSON-loaded from services.json
type ServicesConfig struct {
	Services []ServiceInfo `json:"services"`
}

// ServiceInfo represents information about a service
// Controls flags define whether UI actions are permitted.
// run_path/run_env allow custom process start.
type ServiceInfo struct {
	Port         int               `json:"port"`
	Name         string            `json:"name"`
	Link         string            `json:"link,omitempty"`
	Image        string            `json:"image,omitempty"`
	ShowPort     bool              `json:"show_port,omitempty"`
	ServiceName  string            `json:"service_name,omitempty"`
	SystemdName  string            `json:"systemd_name,omitempty"`
	Controls     bool              `json:"controls,omitempty"`
	ControlsRun  bool              `json:"controls_run,omitempty"`
	ControlsShut bool              `json:"controls_shut,omitempty"`
	RunPath      string            `json:"run_path,omitempty"`
	RunEnv       map[string]string `json:"run_env,omitempty"`
}

// Service is the rendered status entry for the UI.
type Service struct {
	Port         int
	Name         string
	Link         string
	Image        string
	ShowPort     bool
	SystemdName  string
	IsSystemd    bool
	Active       bool
	Controls     bool
	ControlsRun  bool
	ControlsShut bool
}
