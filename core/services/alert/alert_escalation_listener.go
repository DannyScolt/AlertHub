package alert

import (
	"context"
	"encoding/json"
	"log"
	"time"

	alertRepo "alerthub/core/repository/alert"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EscalationListener struct {
	db      *pgxpool.Pool
	service EscalationService
	channel string
}

func NewEscalationListener(db *pgxpool.Pool, service EscalationService) *EscalationListener {
	return &EscalationListener{db: db, service: service, channel: alertRepo.ChannelAlertCreated}
}

func (l *EscalationListener) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := l.listen(ctx); err != nil && ctx.Err() == nil {
			log.Printf("alert escalation listener disconnected: %v", err)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = minDuration(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second
	}
}

func (l *EscalationListener) listen(ctx context.Context) error {
	conn, err := l.db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `LISTEN `+l.channel); err != nil {
		return err
	}

	for ctx.Err() == nil {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		l.handleNotification(ctx, notification)
	}
	return ctx.Err()
}

func (l *EscalationListener) handleNotification(ctx context.Context, notification *pgconn.Notification) {
	if notification == nil {
		return
	}
	var payload alertNotificationPayload
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		log.Printf("invalid escalation notification payload: %v", err)
		return
	}
	alertID, err := uuid.Parse(payload.AlertID)
	if err != nil {
		log.Printf("invalid escalation notification alert_id: %v", err)
		return
	}
	if _, err := l.service.HandleNewAlert(ctx, EscalateInput{AlertID: alertID}); err != nil {
		log.Printf("handle alert escalation failed: %v", err)
	}
}
