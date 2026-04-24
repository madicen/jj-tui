package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRetryAfterWait_Seconds(t *testing.T) {
	if d := parseRetryAfterWait("3"); d != 3*time.Second {
		t.Fatalf("got %v", d)
	}
}

func TestParseRetryAfterWait_HTTPDate(t *testing.T) {
	when := time.Now().UTC().Add(2 * time.Second).Format(http.TimeFormat)
	d := parseRetryAfterWait(when)
	if d < time.Millisecond || d > 5*time.Second {
		t.Fatalf("unexpected duration %v", d)
	}
}

func TestClient_Complete_Retries429(t *testing.T) {
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 2 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "ok"}},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-key", "m1", 10*time.Second)
	out, err := c.Complete(context.Background(), "s", "u")
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
	if n != 2 {
		t.Fatalf("attempts=%d", n)
	}
}

func TestClient_Complete_NoRetryOn401(t *testing.T) {
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "k", "m", time.Second)
	_, err := c.Complete(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("got %v", err)
	}
	if n != 1 {
		t.Fatalf("attempts=%d want 1", n)
	}
}
