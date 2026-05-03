package alert

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ChannelAlertCreated = "alert_channel"

type Notifier interface {
	NotifyAlertCreated(ctx context.Context, clientID, alertID uuid.UUID) error
}

type postgresNotifier struct{ db *pgxpool.Pool }

func NewNotifier(db *pgxpool.Pool) Notifier { return &postgresNotifier{db: db} }

func (n *postgresNotifier) NotifyAlertCreated(ctx context.Context, clientID, alertID uuid.UUID) error {
	payload, err := json.Marshal(map[string]string{
		"client_id": clientID.String(),
		"alert_id":  alertID.String(),
	})
	if err != nil {
		return err
	}
	_, err = n.db.Exec(ctx, `SELECT pg_notify($1, $2)`, ChannelAlertCreated, string(payload))
	return err
}
