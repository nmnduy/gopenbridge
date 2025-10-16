package server

import (
	"encoding/json"
	"log"
	"net/http"
	"openbridge/config"
	"openbridge/proxy"
	"openbridge/templates"
	"strconv"
)

// StartServer starts HTTP server on given address.
// StartServer starts HTTP server using configuration.
func StartServer(cfg *config.Config) error {
	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	// Load HTML templates
	if err := templates.LoadTemplates("templates/*.tmpl"); err != nil {
		log.Printf("Failed to load templates: %v", err)
		return err
	}
	mux := http.NewServeMux()

	// Root endpoint serves rendered homepage template
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		data := templates.HomepageData{Host: cfg.Host, Port: cfg.Port, Model: cfg.Model}
		if err := templates.RenderHomepage(w, data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	})

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy", "model": cfg.Model})
	})

	// Chat proxy for messages endpoint (Anthropic -> OpenAI)
	chatProxy := proxy.NewChatProxy(cfg)
	mux.Handle("/v1/messages", chatProxy)

	// Start HTTP server
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, mux)
}
