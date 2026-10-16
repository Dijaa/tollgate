// Command echo is a tiny upstream API used to try the gateway locally.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "hello from the upstream API",
			"path":    r.URL.Path,
			"time":    time.Now().Format(time.RFC3339),
		})
	})
	log.Println("echo API listening on :9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
