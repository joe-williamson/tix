// Package config handles loading of breakglass profiles and Jira credentials.
package config

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed bg_profiles.yaml
var defaultProfilesYAML []byte

// Creds holds Jira authentication credentials.
type Creds struct {
	BaseURL string
	User    string
	Token   string
}

// Profile holds a single breakglass profile entry.
type Profile struct {
	Provider      string `yaml:"provider"`
	Group         string `yaml:"group"`
	User          string `yaml:"user"`
	Hours         int    `yaml:"hours"`
	Env           string `yaml:"env"`
	Project       string `yaml:"project"`
	Namespace     string `yaml:"namespace"`
	SummaryPrefix string `yaml:"summary_prefix"`
}

type profilesFile struct {
	Defaults Profile            `yaml:"defaults"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// DefaultProfilesPath returns the default path for the profiles YAML file.
func DefaultProfilesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bg_profiles.yaml")
}

// Load reads defaults and named profiles from ~/.bg_profiles.yaml (or $BG_PROFILES).
// Falls back to the embedded default template if the file doesn't exist.
func Load() (Profile, map[string]Profile, error) {
	path := os.Getenv("BG_PROFILES")
	if path == "" {
		path = DefaultProfilesPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Profile{}, nil, fmt.Errorf("cannot read %s: %w", path, err)
		}
		// File not found — use embedded template.
		data = defaultProfilesYAML
	}

	var f profilesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return Profile{}, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return f.Defaults, f.Profiles, nil
}

// UserFromEmail returns the local part of an email address (before '@').
// If the string has no '@', it is returned unchanged.
func UserFromEmail(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}

// Resolve merges defaults → named profile → CLI overrides into a final Profile.
func Resolve(name string, defaults Profile, profiles map[string]Profile, overrides Profile) (Profile, error) {
	p, ok := profiles[name]
	if !ok {
		names := make([]string, 0, len(profiles))
		for k := range profiles {
			names = append(names, k)
		}
		return Profile{}, fmt.Errorf("unknown profile %q\navailable: %s", name, strings.Join(names, ", "))
	}

	// Apply defaults for zero values.
	if p.User == "" {
		p.User = defaults.User
	}
	if p.Hours == 0 {
		p.Hours = defaults.Hours
	}
	if p.Env == "" {
		p.Env = defaults.Env
	}

	// Apply CLI overrides for non-empty values.
	if overrides.User != "" {
		p.User = overrides.User
	}
	if overrides.Hours != 0 {
		p.Hours = overrides.Hours
	}
	if overrides.Group != "" {
		p.Group = overrides.Group
	}
	if overrides.Project != "" {
		p.Project = overrides.Project
	}
	if overrides.Namespace != "" {
		p.Namespace = overrides.Namespace
	}
	if overrides.Provider != "" {
		p.Provider = overrides.Provider
	}

	for _, req := range []struct{ name, val string }{
		{"provider", p.Provider},
		{"group", p.Group},
		{"user", p.User},
	} {
		if req.val == "" {
			return Profile{}, fmt.Errorf("profile %q missing required field: %s", name, req.name)
		}
	}

	return p, nil
}

// LoadJiraCreds reads credentials from ~/.jira_config (INI format).
func LoadJiraCreds() (Creds, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Creds{}, fmt.Errorf("home dir: %w", err)
	}
	path := filepath.Join(home, ".jira_config")
	f, err := os.Open(path)
	if err != nil {
		return Creds{}, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	baseURL := os.Getenv("JIRA_URL")
	if baseURL == "" {
		baseURL = "https://perzoinc.atlassian.net"
	}
	creds := Creds{BaseURL: baseURL}
	inSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = strings.EqualFold(strings.TrimSpace(line[1:len(line)-1]), "jira")
			continue
		}
		if !inSection {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch strings.ToLower(key) {
		case "user_name", "username", "user", "email":
			if creds.User == "" {
				creds.User = val
			}
		case "token", "api_token", "jira_token":
			if creds.Token == "" {
				creds.Token = val
			}
		}
	}

	if creds.User == "" || creds.Token == "" {
		return Creds{}, fmt.Errorf("user_name and token required in [jira] section of %s", path)
	}
	return creds, nil
}
