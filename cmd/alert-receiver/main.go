package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type WebhookMessage struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON payload
	var msg WebhookMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Log the alerts to stdout
	fmt.Printf("Received Alert (Status: %s) | Receiver: %s\n", msg.Status, msg.Receiver)
	for _, alert := range msg.Alerts {
		fmt.Printf(" - Alert: %s @ %s\n", alert.Labels["alertname"], alert.StartsAt.Format(time.RFC3339))
		fmt.Printf("   Summary: %s\n", alert.Annotations["summary"])
		fmt.Printf("   Description: %s\n", alert.Annotations["description"])
	}
	fmt.Println("------------------------------------------------------")
}

func main() {
	http.HandleFunc("/alerts", webhookHandler)

	port := ":9080"
	log.Printf("Listening for alerts on %s/alerts...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
