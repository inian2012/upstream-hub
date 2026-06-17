package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/worryzyy/upstream-hub/internal/connector"
)

func TestGetUsageStats(t *testing.T) {
	var sawLogStat bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/status":
			writeNewAPIData(t, w, map[string]any{"quota_per_unit": 500000})
		case "/api/user/self":
			if got := r.Header.Get("Cookie"); got != "session=abc" {
				t.Fatalf("Cookie header = %q, want session=abc", got)
			}
			if got := r.Header.Get("New-Api-User"); got != "42" {
				t.Fatalf("New-Api-User header = %q, want 42", got)
			}
			writeNewAPIData(t, w, map[string]any{
				"quota":      3500000,
				"used_quota": 1500000,
			})
		case "/api/log/self/stat":
			sawLogStat = true
			if got := r.Header.Get("Cookie"); got != "session=abc" {
				t.Fatalf("Cookie header = %q, want session=abc", got)
			}
			if got := r.Header.Get("New-Api-User"); got != "42" {
				t.Fatalf("New-Api-User header = %q, want 42", got)
			}
			if got := r.URL.Query().Get("type"); got != "2" {
				t.Fatalf("type query = %q, want 2", got)
			}
			start, err := strconv.ParseInt(r.URL.Query().Get("start_timestamp"), 10, 64)
			if err != nil || start <= 0 {
				t.Fatalf("start_timestamp = %q, want positive unix seconds", r.URL.Query().Get("start_timestamp"))
			}
			end, err := strconv.ParseInt(r.URL.Query().Get("end_timestamp"), 10, 64)
			if err != nil || end < start {
				t.Fatalf("end_timestamp = %q, want unix seconds after start", r.URL.Query().Get("end_timestamp"))
			}
			writeNewAPIData(t, w, map[string]any{"quota": 750000})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stats, err := New().GetUsageStats(context.Background(), &connector.Channel{SiteURL: server.URL}, &connector.AuthSession{
		UserID: "42",
		Cookie: "session=abc",
	})
	if err != nil {
		t.Fatalf("GetUsageStats() error = %v", err)
	}
	if !sawLogStat {
		t.Fatal("GetUsageStats() did not call /api/log/self/stat")
	}
	if stats.TodayActualCost != 1.5 {
		t.Fatalf("TodayActualCost = %v, want 1.5", stats.TodayActualCost)
	}
	if stats.TotalActualCost != 3 {
		t.Fatalf("TotalActualCost = %v, want 3", stats.TotalActualCost)
	}
	if stats.SampledAt.IsZero() {
		t.Fatal("SampledAt is zero")
	}
}

func TestGetBalanceUsesDefaultQuotaPerUnit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/status":
			writeNewAPIData(t, w, map[string]any{"quota_per_unit": 0})
		case "/api/user/self":
			writeNewAPIData(t, w, map[string]any{"quota": 1000000})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	balance, err := New().GetBalance(context.Background(), &connector.Channel{SiteURL: server.URL}, &connector.AuthSession{})
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if balance.Balance != 2 {
		t.Fatalf("Balance = %v, want 2", balance.Balance)
	}
}

func writeNewAPIData(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "",
		"data":    data,
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
