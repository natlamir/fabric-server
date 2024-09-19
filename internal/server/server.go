package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	router *mux.Router
}

func NewServer() *Server {
	s := &Server{
		router: mux.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/api/fabric/options", s.getOptions).Methods("GET")
	s.router.HandleFunc("/api/fabric", s.postFabric).Methods("POST")
}

func (s *Server) Start() error {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		Debug:            true,
	})

	handler := c.Handler(s.router)

	fmt.Println("Fabric server listening at http://localhost:3001")
	return http.ListenAndServe(":3001", handler)
}

func (s *Server) getOptions(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Fetching options from fabric...")
	
	cmd := exec.Command("cmd", "/C", "fabric -l")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create stdout pipe", http.StatusInternalServerError)
		return
	}

	err = cmd.Start()
	if err != nil {
		http.Error(w, "Failed to start fabric command", http.StatusInternalServerError)
		return
	}

	var options []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && line != "Patterns:" {
			options = append(options, line)
		}
	}

	err = cmd.Wait()
	if err != nil {
		http.Error(w, fmt.Sprintf("Fabric command failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (s *Server) postFabric(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input  string `json:"input"`
		Option string `json:"option"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	escapedInput := strings.Replace(req.Input, "\"", "\\\"", -1)
	fabricCommand := fmt.Sprintf("echo %s | fabric -p %s -s", escapedInput, req.Option)
	
	cmd := exec.Command("cmd", "/C", fabricCommand)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create stdout pipe", http.StatusInternalServerError)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		http.Error(w, "Failed to create stderr pipe", http.StatusInternalServerError)
		return
	}

	err = cmd.Start()
	if err != nil {
		http.Error(w, "Failed to start command", http.StatusInternalServerError)
		return
	}

	go func() {
		io.Copy(w, stdout)
	}()

	go func() {
		io.Copy(w, stderr)
	}()

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintf(w, "Command finished with error: %v", err)
	}
}