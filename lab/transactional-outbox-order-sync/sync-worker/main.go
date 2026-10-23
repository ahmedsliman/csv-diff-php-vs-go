// Sync worker - the "messaging service" of the Transactional Outbox pattern.
//
// It polls each PHP service's outbox table for unprocessed events and applies
// them to the peer service. Because every order + outbox write happened in one
// local transaction (see php-service/index.php), there is no dual-write
// problem. Delivery is at-least-once; the receiver is idempotent (UNIQUE
// order_number + ON DUPLICATE KEY UPDATE) so redelivery after a crash is safe.
//
// See DESIGN.md for the full design.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

var (
	client = &http.Client{Timeout: 10 * time.Second}

	pollInterval = durationEnv("POLL_INTERVAL", 3*time.Second)
	serviceA     = env("SERVICE_A_BASE_URL", "http://service-a")
	serviceB     = env("SERVICE_B_BASE_URL", "http://service-b")
)

type outboxEntry struct {
	ID                 int     `json:"id"`
	AggregateType      string  `json:"aggregate_type"`
	AggregateID        string  `json:"aggregate_id"`
	Type               string  `json:"type"`
	Payload            []byte  `json:"-"`
	CreatedAt          string  `json:"created_at"`
	ProcessedAt        *string `json:"processed_at"`
	ProcessingAttempts int     `json:"processing_attempts"`
}

type outboxList struct {
	Outbox []outboxEntry `json:"outbox"`
}

// UnmarshalJSON decodes payload as raw JSON bytes so we can forward it
// verbatim to the peer without interpreting it.
func (e *outboxEntry) UnmarshalJSON(data []byte) error {
	aux := struct {
		ID                 int             `json:"id"`
		AggregateType      string          `json:"aggregate_type"`
		AggregateID        string          `json:"aggregate_id"`
		Type               string          `json:"type"`
		Payload            json.RawMessage `json:"payload"`
		CreatedAt          string          `json:"created_at"`
		ProcessedAt        *string         `json:"processed_at"`
		ProcessingAttempts int             `json:"processing_attempts"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.ID = aux.ID
	e.AggregateType = aux.AggregateType
	e.AggregateID = aux.AggregateID
	e.Type = aux.Type
	e.Payload = []byte(aux.Payload)
	e.CreatedAt = aux.CreatedAt
	e.ProcessedAt = aux.ProcessedAt
	e.ProcessingAttempts = aux.ProcessingAttempts
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("sync-worker started: %s <-> %s, poll interval %s",
		serviceA, serviceB, pollInterval)

	for {
		if err := syncDirection(serviceA, serviceB, "service-a", "service-b"); err != nil {
			log.Printf("sync service-a -> service-b: %v", err)
		}
		if err := syncDirection(serviceB, serviceA, "service-b", "service-a"); err != nil {
			log.Printf("sync service-b -> service-a: %v", err)
		}
		time.Sleep(pollInterval)
	}
}

// syncDirection fetches unprocessed outbox entries from one service and
// delivers each to the other service's order endpoint.
func syncDirection(fromBase, toBase, fromService, toService string) error {
	outboxURL := fromBase + "/outbox?processed=0"
	ordersURL := toBase + "/orders"

	resp, err := client.Get(outboxURL)
	if err != nil {
		return fmt.Errorf("fetch outbox from %s: %w", outboxURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("outbox endpoint returned %d: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var list outboxList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return fmt.Errorf("decode outbox response: %w", err)
	}

	for _, entry := range list.Outbox {
		if err := deliver(ordersURL, entry); err != nil {
			log.Printf("outbox id=%d: deliver %s -> %s failed: %v (retrying)",
				entry.ID, fromService, toService, err)
			continue
		}
		processURL := fmt.Sprintf("%s/outbox/%d/process", fromBase, entry.ID)
		if err := markProcessed(processURL); err != nil {
			// Delivered but not marked: the entry will be redelivered. The
			// receiver's ON DUPLICATE KEY makes this idempotent.
			log.Printf("outbox id=%d: delivered but not marked processed: %v",
				entry.ID, err)
			continue
		}
		log.Printf("outbox id=%d type=%s order=%s synced %s -> %s",
			entry.ID, entry.Type, entry.AggregateID, fromService, toService)
	}
	return nil
}

// deliver posts an outbox event's payload to the peer's order endpoint.
// X-Synced: true tells the peer to store the order without appending a new
// outbox event, preventing a sync loop.
func deliver(ordersURL string, entry outboxEntry) error {
	req, err := http.NewRequest(http.MethodPost, ordersURL,
		bytes.NewReader(entry.Payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Synced", "true")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("order endpoint returned %d", resp.StatusCode)
	}
	return nil
}

// markProcessed tells the source service to mark the outbox entry processed.
func markProcessed(url string) error {
	req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mark processed returned %d", resp.StatusCode)
	}
	return nil
}
