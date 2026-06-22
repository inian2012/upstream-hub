// Package connector defines the common upstream connector interface.
package connector

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ChannelType identifies the upstream API family.
type ChannelType string

const (
	TypeNewAPI  ChannelType = "newapi"
	TypeSub2API ChannelType = "sub2api"
)

// Channel is the decrypted channel configuration passed to connectors.
type Channel struct {
	ID               uint
	Name             string
	Type             ChannelType
	SiteURL          string
	Username         string
	Password         string
	TurnstileEnabled bool
	TurnstileToken   string
}

// AuthSession is the plain session material used by connectors.
type AuthSession struct {
	UserID      string
	AccessToken string
	Cookie      string
	CSRFToken   string
	ExpiresAt   time.Time
}

// BalanceResult is one balance collection result in upstream account units.
type BalanceResult struct {
	Balance   float64
	SampledAt time.Time
}

// RateResult is one visible upstream group/model multiplier record.
type RateResult struct {
	ModelName       string
	Description     string
	Ratio           float64
	RatioLabel      string
	CompletionRatio float64
}

// UsageStatsResult is an optional upstream-provided consumption snapshot.
// Costs are in the upstream account unit before recharge-ratio conversion.
type UsageStatsResult struct {
	TodayActualCost float64
	TotalActualCost float64
	SampledAt       time.Time
}

// Connector is the common interface every upstream implementation provides.
type Connector interface {
	GetTurnstileSiteKey(ctx context.Context, channel *Channel) (string, error)
	Login(ctx context.Context, channel *Channel) (*AuthSession, error)
	CheckAuth(ctx context.Context, channel *Channel, session *AuthSession) error
	GetBalance(ctx context.Context, channel *Channel, session *AuthSession) (*BalanceResult, error)
	GetRates(ctx context.Context, channel *Channel, session *AuthSession) ([]RateResult, error)
}

// UsageStatsProvider is implemented by connectors whose upstream exposes a
// real usage ledger, avoiding balance-delta undercount when users recharge.
type UsageStatsProvider interface {
	GetUsageStats(ctx context.Context, channel *Channel, session *AuthSession) (*UsageStatsResult, error)
}

// Factory constructs a fresh Connector.
type Factory func() Connector

var (
	mu       sync.RWMutex
	registry = map[ChannelType]Factory{}
)

// Register associates a connector factory with a channel type.
func Register(t ChannelType, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[t] = f
}

// For returns a new connector for a channel type.
func For(t ChannelType) (Connector, error) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := registry[t]
	if !ok {
		return nil, fmt.Errorf("connector %q is not registered (did you forget the blank import?)", t)
	}
	return f(), nil
}
