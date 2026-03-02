package event

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"whatsapp-ruok/db"
	"whatsapp-ruok/wasender"
)

var israelTZ *time.Location

func init() {
	var err error
	israelTZ, err = time.LoadLocation("Asia/Jerusalem")
	if err != nil {
		log.Fatalf("failed to load Asia/Jerusalem timezone: %v", err)
	}
}

const triggerMessage = "בוט תבדוק שכולם בסדר"
const stopTriggerMessage = "בוט אפשר לעצור"

type activeEvent struct {
	eventID int
	cancel  context.CancelFunc
}

type Manager struct {
	mu       sync.Mutex
	active   map[string]*activeEvent
	db       *db.DB
	client   *wasender.Client
	botPhone string
}

func NewManager(database *db.DB, client *wasender.Client, botPhone string) *Manager {
	return &Manager{
		active:   make(map[string]*activeEvent),
		db:       database,
		client:   client,
		botPhone: normalizePhone(botPhone),
	}
}

func (m *Manager) ResumeActiveEvents(ctx context.Context) error {
	events, err := m.db.GetAllActiveEvents(ctx)
	if err != nil {
		return err
	}
	for _, e := range events {
		m.startTicker(e.ID, e.GroupJID)
		log.Printf("resumed event %d for group %s", e.ID, e.GroupJID)
	}
	return nil
}

func (m *Manager) HandleGroupMessage(groupJID, participantLID, cleanedPhone, messageBody string, fromMe bool) {
	if fromMe {
		return
	}

	phone := normalizePhone(cleanedPhone)
	startEventMsg := strings.TrimSpace(messageBody) == triggerMessage
	stopEventMsg := strings.TrimSpace(messageBody) == stopTriggerMessage
	msgType := "start"
	if stopEventMsg {
		msgType = "stop"
	}
	if startEventMsg || stopEventMsg {
		m.handleTrigger(groupJID, participantLID, phone, msgType)
		return
	}

	m.handleResponse(groupJID, phone, messageBody)
}

func (m *Manager) handleTrigger(groupJID, adminLID, adminPhone string, msgType string) {
	ctx := context.Background()

	isAdmin, err := m.checkAdmin(ctx, groupJID, adminLID, adminPhone)
	if err != nil {
		log.Printf("failed to check admin status for %s: %v", adminPhone, err)
		return
	}
	if !isAdmin {
		log.Printf("non-admin %s tried to trigger event in %s", adminPhone, groupJID)
		return
	}

	m.cancelExisting(groupJID)

	if err := m.db.DeactivateGroupEvents(ctx, groupJID); err != nil {
		log.Printf("failed to deactivate old events: %v", err)
	}

	if msgType == "start" {
		eventID, err := m.db.CreateEvent(ctx, groupJID, adminLID, adminPhone)
		if err != nil {
			log.Printf("failed to create event: %v", err)
			return
		}

		participants, err := m.fetchAndStoreParticipants(ctx, eventID, groupJID, adminPhone)
		if err != nil {
			log.Printf("failed to fetch participants: %v", err)
			return
		}

		now := time.Now().In(israelTZ)
		announcement := fmt.Sprintf("@%s שאל ב-%s שכולם בסדר - נא לענות 🌟", adminPhone, now.Format("15:04"))

		if err := m.client.SendMessage(ctx, wasender.SendMessageRequest{
			To:       groupJID,
			Text:     announcement,
			Mentions: []string{adminPhone + "@s.whatsapp.net"},
		}); err != nil {
			log.Printf("failed to send announcement: %v", err)
		}

		log.Printf("event %d started in %s with %d participants", eventID, groupJID, len(participants))
		m.startTicker(eventID, groupJID)
	}
}

func (m *Manager) handleResponse(groupJID, phone, messageBody string) {
	ctx := context.Background()

	evt, err := m.db.GetActiveEvent(ctx, groupJID)
	if err != nil {
		return
	}

	if phone == m.botPhone {
		return
	}

	now := time.Now().In(israelTZ)
	updated, err := m.db.RecordResponse(ctx, evt.ID, phone, messageBody, now)
	if err != nil {
		log.Printf("failed to record response: %v", err)
		return
	}
	if !updated {
		return
	}

	unresponded, err := m.db.GetUnrespondedCount(ctx, evt.ID)
	if err != nil {
		log.Printf("failed to get unresponded count: %v", err)
		return
	}

	if unresponded == 0 {
		m.completeEvent(ctx, evt.ID, groupJID)
	}
}

func (m *Manager) completeEvent(ctx context.Context, eventID int, groupJID string) {
	m.cancelExisting(groupJID)

	if err := m.db.CompleteEvent(ctx, eventID); err != nil {
		log.Printf("failed to complete event: %v", err)
	}

	participants, err := m.db.GetParticipants(ctx, eventID)
	if err != nil {
		log.Printf("failed to get participants for completion: %v", err)
		return
	}

	lines := []string{"🎉 כולם ענו - תודה רבה! 🌟", ""}
	var mentions []string

	for _, p := range participants {
		ts := ""
		if p.RespondedAt != nil {
			ts = p.RespondedAt.In(israelTZ).Format("15:04")
		}
		lines = append(lines, fmt.Sprintf("🥳 - @%s ענה ב-%s 🌈 ׳%s׳ 🎀", p.Phone, ts, p.ResponseText))
		mentions = append(mentions, p.Phone+"@s.whatsapp.net")
	}

	if err := m.client.SendMessage(ctx, wasender.SendMessageRequest{
		To:       groupJID,
		Text:     strings.Join(lines, "\n"),
		Mentions: mentions,
	}); err != nil {
		log.Printf("failed to send completion message: %v", err)
	}

	log.Printf("event %d completed in %s", eventID, groupJID)
}

func (m *Manager) startTicker(eventID int, groupJID string) {
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.active[groupJID] = &activeEvent{eventID: eventID, cancel: cancel}
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.sendStatusUpdate(ctx, eventID, groupJID)
			}
		}
	}()
}

func (m *Manager) sendStatusUpdate(ctx context.Context, eventID int, groupJID string) {
	participants, err := m.db.GetParticipants(ctx, eventID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("failed to get participants for status: %v", err)
		return
	}

	lines := []string{"סטטוס נוכחי:", ""}
	var mentions []string

	for _, p := range participants {
		if p.Responded && p.RespondedAt != nil {
			ts := p.RespondedAt.In(israelTZ).Format("15:04")
			lines = append(lines, fmt.Sprintf("🟢 - @%s שלח.ה תגובה ב-%s '%s'", p.Phone, ts, p.ResponseText))
		} else {
			lines = append(lines, fmt.Sprintf("🟡 - @%s עדיין לא שלח.ה תגובה ⏳", p.Phone))
		}
		mentions = append(mentions, p.Phone+"@s.whatsapp.net")
	}

	if err := m.client.SendMessage(ctx, wasender.SendMessageRequest{
		To:       groupJID,
		Text:     strings.Join(lines, "\n"),
		Mentions: mentions,
	}); err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("failed to send status update: %v", err)
	}
}

func (m *Manager) cancelExisting(groupJID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.active[groupJID]; ok {
		existing.cancel()
		delete(m.active, groupJID)
	}
}

func (m *Manager) checkAdmin(ctx context.Context, groupJID, participantLID, participantPhone string) (bool, error) {
	meta, err := m.client.GetGroupMetadata(ctx, groupJID)
	if err != nil {
		return false, err
	}

	lidNum := extractPhone(participantLID)

	for _, p := range meta.Participants {
		if !p.IsAdmin && !p.IsSuperAdmin {
			continue
		}
		pNum := extractPhone(p.JID)
		if pNum == participantPhone || pNum == lidNum {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) fetchAndStoreParticipants(ctx context.Context, eventID int, groupJID, adminPhone string) ([]db.Participant, error) {
	groupParticipants, err := m.client.GetGroupParticipants(ctx, groupJID)
	if err != nil {
		return nil, fmt.Errorf("get group participants: %w", err)
	}

	for _, gp := range groupParticipants {
		phone, err := m.resolvePhone(ctx, gp.ID)
		if err != nil {
			log.Printf("skipping participant %s (failed to resolve LID): %v", gp.ID, err)
			continue
		}

		norm := normalizePhone(phone)
		if norm == m.botPhone || norm == adminPhone {
			continue
		}

		if err := m.db.AddParticipant(ctx, eventID, gp.ID, norm); err != nil {
			log.Printf("failed to add participant %s: %v", norm, err)
		}
	}

	return m.db.GetParticipants(ctx, eventID)
}

func (m *Manager) resolvePhone(ctx context.Context, participantID string) (string, error) {
	if strings.HasSuffix(participantID, "@s.whatsapp.net") {
		return extractPhone(participantID), nil
	}
	if strings.HasSuffix(participantID, "@lid") {
		pn, err := m.client.ResolvePhoneFromLID(ctx, participantID)
		if err != nil {
			return "", err
		}
		return extractPhone(pn), nil
	}
	return extractPhone(participantID), nil
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	phone = strings.TrimSuffix(phone, "@lid")
	return phone
}

func extractPhone(jid string) string {
	jid = strings.TrimSpace(jid)
	jid = strings.Split(jid, "@")[0]
	jid = strings.TrimPrefix(jid, "+")
	return jid
}
