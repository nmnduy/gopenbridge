package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds application configuration.
type Config struct {
	APIKey    string // API key for authentication
	BaseURL   string // Base URL for API requests
	Model     string // Model identifier
	MaxTokens int    // Maximum output tokens
	Host      string // Server host
	Port      int    // Server port
}

// LoadConfig loads configuration from file, environment, or defaults.
func LoadConfig() (*Config, error) {
	// Set defaults
	cfg := &Config{
		APIKey:    "",
		BaseURL:   "https://router.huggingface.co/v1",
		Model:     "moonshotai/Kimi-K2-Instruct-0905:groq",
		MaxTokens: 16384,
		Host:      "0.0.0.0",
		Port:      8323,
	}
	// Override with environment variables
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("MAX_OUTPUT_TOKENS"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			cfg.MaxTokens = iv
		}
	}
	if v := os.Getenv("HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			cfg.Port = iv
		}
	}
	// Load from config file if available
	if path := findConfigFile(); path != "" {
		if fileCfg, err := parseYAMLFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not load config file %s: %v\n", path, err)
		} else {
			for k, v := range fileCfg {
				switch k {
				case "api_key":
					cfg.APIKey = v
				case "base_url":
					cfg.BaseURL = v
				case "model":
					cfg.Model = v
				case "max_tokens":
					if iv, err := strconv.Atoi(v); err == nil {
						cfg.MaxTokens = iv
					}
				case "host":
					cfg.Host = v
				case "port":
					if iv, err := strconv.Atoi(v); err == nil {
						cfg.Port = iv
					}
				}
			}
		}
	}
	// Fallback to Hugging Face token if APIKey not set
	if cfg.APIKey == "" {
		if home, err := os.UserHomeDir(); err == nil {
			hfPath := filepath.Join(home, ".huggingface", "token")
			if data, err := os.ReadFile(hfPath); err == nil {
				if token := strings.TrimSpace(string(data)); token != "" {
					cfg.APIKey = token
				}
			}
		}
	}
	return cfg, nil
}

// findConfigFile searches for a YAML config file in standard locations.
func findConfigFile() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"openbridge.yaml",
		"openbridge.yml",
		filepath.Join(home, ".openbridge.yaml"),
		filepath.Join(home, ".openbridge.yml"),
		filepath.Join(home, ".config", "openbridge", "config.yaml"),
		filepath.Join(home, ".config", "openbridge", "config.yml"),
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// parseYAMLFile loads simple key:value pairs from a YAML file.
func parseYAMLFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	res := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			val = strings.Trim(val, `"'`)
			res[key] = val
		}
	}
	if err := scanner.Err(); err != nil {
		return res, err
	}
	return res, nil
}

// IsUsingDefaults returns true if config model and base URL match defaults.
func IsUsingDefaults(cfg *Config) bool {
	return cfg.BaseURL == "https://router.huggingface.co/v1" &&
		cfg.Model == "moonshotai/Kimi-K2-Instruct-0905:groq"
}

// PrintConfigInfo prints tips when defaults are used and shows config file path.
func PrintConfigInfo(cfg *Config) {
	if IsUsingDefaults(cfg) {
		fmt.Println("📝 You're using the default model configuration.")
		fmt.Println("💡 You can customize by creating openbridge.yaml in:")
		fmt.Println("   - Current directory: ./openbridge.yaml")
		fmt.Println("   - Home directory: ~/.openbridge.yaml")
		fmt.Println("   - Config directory: ~/.config/openbridge/config.yaml")
		fmt.Println()
		fmt.Println("Example openbridge.yaml:")
		fmt.Println("---")
		fmt.Println("api_key: your-api-key-here")
		fmt.Println("base_url: https://api.openai.com/v1")
		fmt.Println("model: gpt-4")
		fmt.Println("max_tokens: 4096")
		fmt.Println()
	}
	if p := findConfigFile(); p != "" {
		fmt.Printf("📋 Using config from: %s\n", p)
	} else {
		fmt.Println("📋 No config file found, using defaults and environment variables")
	}
}
