package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
)

var services = []string{"mediamtx", "stream.service", "monitor-http.service"}

type serviceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]serviceStatus, len(services))
	for i, svc := range services {
		err := exec.Command("systemctl", "is-active", "--quiet", svc).Run()
		result[i] = serviceStatus{Name: svc, Active: err == nil}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	args := []string{"--no-pager", "-n", "150", "--output=short"}
	for _, svc := range services {
		args = append(args, "-u", svc)
	}
	out, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		http.Error(w, "Failed to fetch logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(out)
}

func main() {
	http.HandleFunc("/api/status", statusHandler)
	http.HandleFunc("/api/logs", logsHandler)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	addr := ":80"
	log.Printf("Serving on http://0.0.0.0%s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
