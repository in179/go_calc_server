package server

import (
	v1 "calculator/internal/api/v1"
	"log"
	"net/http"
	"path/filepath"
)

func StartServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", v1.SubmitExpression)
	mux.HandleFunc("/api/v1/expressions/", v1.GetExpression)
	mux.HandleFunc("/api/v1/expressions", v1.ListExpressions)
	mux.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			v1.GetTask(w, r)
		} else if r.Method == http.MethodPost {
			v1.PostTaskResult(w, r)
		} else {
			http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	webDir, _ := filepath.Abs("./web")
	fs := http.FileServer(http.Dir(webDir))
	mux.Handle("/", fs)

	log.Println("Orchestrator server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
