package alert

import (
	"context"
	"encoding/json"
	"log"
	"time"

	domain "alerthub/core/domain/alert"
	alertRepo "alerthub/core/repository/alert"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertDispatcher interface {
	Dispatch(alert domain.Alert)
}

type AlertListener struct {
	db         *pgxpool.Pool
	repo       alertRepo.AlertRepository
	dispatcher AlertDispatcher
	channel    string
}

type alertNotificationPayload struct {
	ClientID string `json:"client_id"`
	AlertID  string `json:"alert_id"`
}

func NewAlertListener(db *pgxpool.Pool, repo alertRepo.AlertRepository, dispatcher AlertDispatcher) *AlertListener {
	return &AlertListener{db: db, repo: repo, dispatcher: dispatcher, channel: alertRepo.ChannelAlertCreated}
}

func (l *AlertListener) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := l.listen(ctx); err != nil && ctx.Err() == nil {
			log.Printf("alert listener disconnected: %v", err)
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

func (l *AlertListener) listen(ctx context.Context) error {
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

func (l *AlertListener) handleNotification(ctx context.Context, notification *pgconn.Notification) {
	if notification == nil {
		return
	}
	var payload alertNotificationPayload
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		log.Printf("invalid alert notification payload: %v", err)
		return
	}
	alertID, err := uuid.Parse(payload.AlertID)
	if err != nil {
		log.Printf("invalid alert notification alert_id: %v", err)
		return
	}
	alert, err := l.repo.FindByID(ctx, alertID)
	if err != nil {
		log.Printf("fetch alert from notification failed: %v", err)
		return
	}
	if payload.ClientID != "" && payload.ClientID != alert.ClientID.String() {
		log.Printf("alert notification client mismatch: payload=%s alert=%s", payload.ClientID, alert.ClientID.String())
		return
	}
	l.dispatcher.Dispatch(alert)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
