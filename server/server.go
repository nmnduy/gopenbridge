package server

import (
	"encoding/json"
	"gopenbridge/config"
	"gopenbridge/proxy"
	"log"
	"net/http"
	"strconv"
)

// StartServer starts HTTP server on given address.
// StartServer starts HTTP server using configuration.
func StartServer(cfg *config.Config) error {
	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)

	mux := http.NewServeMux()

	// Root endpoint serves rendered homepage template
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		html := `
<!DOCTYPE html>
<html>
<head><title>gopenbridge</title><style>
body { font-family: Arial; max-width: 800px; margin: 40px auto; padding: 20px; }
.status { background: #e3f2fd; padding: 20px; border-radius: 8px; }
</style></head>
<body>
<h1>🌉 gopenbridge</h1>
<div class="status">
    <h2>Status: Running</h2>
    <p>Proxy listening on ` + cfg.Host + `:` + strconv.Itoa(cfg.Port) + `</p>
    <p>Model: ` + cfg.Model + `</p>
</div>
</body>
</html>`
		w.Write([]byte(html))
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
