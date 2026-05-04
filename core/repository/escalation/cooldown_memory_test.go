package escalation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryCooldownClaimsFirstTime(t *testing.T) {
	store := NewMemoryCooldownStore(time.Now)
	claimed, err := store.ClaimEscalation(context.Background(), cooldownKey(), time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to win")
	}
}

func TestMemoryCooldownRejectsImmediateSecondClaim(t *testing.T) {
	store := NewMemoryCooldownStore(time.Now)
	key := cooldownKey()
	_, _ = store.ClaimEscalation(context.Background(), key, time.Minute)

	claimed, err := store.ClaimEscalation(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claimed {
		t.Fatal("expected second immediate claim to lose")
	}
}

func TestMemoryCooldownAllowsClaimAfterTTL(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	store := NewMemoryCooldownStore(func() time.Time { return now })
	key := cooldownKey()
	_, _ = store.ClaimEscalation(context.Background(), key, time.Minute)
	now = now.Add(time.Minute + time.Second)

	claimed, err := store.ClaimEscalation(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !claimed {
		t.Fatal("expected claim after ttl to win")
	}
}

func TestMemoryCooldownIsolatesDifferentKeys(t *testing.T) {
	store := NewMemoryCooldownStore(time.Now)
	first := cooldownKey()
	second := CooldownKey{DeviceID: uuid.New(), AlertType: first.AlertType}
	_, _ = store.ClaimEscalation(context.Background(), first, time.Minute)

	claimed, err := store.ClaimEscalation(context.Background(), second, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !claimed {
		t.Fatal("expected different key to win")
	}
}

func TestMemoryCooldownConcurrentClaimsExactlyOneWins(t *testing.T) {
	store := NewMemoryCooldownStore(time.Now)
	key := cooldownKey()
	var wg sync.WaitGroup
	wins := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := store.ClaimEscalation(context.Background(), key, time.Minute)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}
			wins <- claimed
		}()
	}
	wg.Wait()
	close(wins)

	count := 0
	for claimed := range wins {
		if claimed {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", count)
	}
}

func cooldownKey() CooldownKey {
	return CooldownKey{DeviceID: uuid.New(), AlertType: "temperature"}
}
