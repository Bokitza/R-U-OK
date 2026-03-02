package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"

	"whatsapp-ruok/event"
)

const (
	maxBodySize    = 1 << 20 // 1 MB
	maxConcurrent  = 50
	maxMessageLen  = 24
)

type WebhookHandler struct {
	secret  string
	manager *event.Manager
	sem     chan struct{}
}

type WebhookPayload struct {
	Event     string          `json:"event"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type GroupMessageData struct {
	Messages GroupMessage `json:"messages"`
}

type GroupMessage struct {
	Key         MessageKey `json:"key"`
	MessageBody string     `json:"messageBody"`
}

type MessageKey struct {
	ID                   string `json:"id"`
	FromMe               bool   `json:"fromMe"`
	RemoteJID            string `json:"remoteJid"`
	Participant          string `json:"participant"`
	ParticipantPn        string `json:"participantPn"`
	CleanedParticipantPn string `json:"cleanedParticipantPn"`
}

func NewWebhookHandler(secret string, manager *event.Manager) *WebhookHandler {
	return &WebhookHandler{
		secret:  secret,
		manager: manager,
		sem:     make(chan struct{}, maxConcurrent),
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sig := r.Header.Get("X-Webhook-Signature")
	if subtle.ConstantTimeCompare([]byte(sig), []byte(h.secret)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"received": true})

	select {
	case h.sem <- struct{}{}:
		go func() {
			defer func() { <-h.sem }()
			h.processEvent(payload)
		}()
	default:
		log.Println("dropping webhook event: concurrency limit reached")
	}
}

func (h *WebhookHandler) processEvent(payload WebhookPayload) {
	switch payload.Event {
	case "messages-group.received":
		var data GroupMessageData
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			log.Printf("failed to parse group message: %v", err)
			return
		}

		body := data.Messages.MessageBody
		if len(body) > maxMessageLen {
			body = body[:maxMessageLen]
		}

		h.manager.HandleGroupMessage(
			data.Messages.Key.RemoteJID,
			data.Messages.Key.Participant,
			data.Messages.Key.CleanedParticipantPn,
			body,
			data.Messages.Key.FromMe,
		)
	}
}
