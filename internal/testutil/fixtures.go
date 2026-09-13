package testutil

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"database/sql"
	"github.com/google/uuid"
	"time"
)

// ObjectMother supplies named, valid examples. Every call returns fresh objects.
type ObjectMother struct{}

func (ObjectMother) User() *user.User {
	return &user.User{Login: "alice", PasswordHash: "fixture-hash"}
}
func (ObjectMother) Target() *target.Target {
	return &target.Target{ID: uuid.New(), Endpoint: "https://example.test/health", Config: target.HTTPConfig{Method: "GET"}, IsActive: true, ProbeIntervalSec: 30}
}
func (m ObjectMother) Contact() *alert.Contact {
	return &alert.Contact{ID: uuid.New(), User: m.User(), Name: "Operations", Type: alert.ContactTypeTelegram, Config: alert.TelegramContactConfig{ChatID: 42, BotToken: "test-token"}, IsActive: true}
}
func (m ObjectMother) Window(active bool) maintenance.MaintenanceWindow {
	return maintenance.MaintenanceWindow{ID: uuid.New(), User: m.User(), Title: "Maintenance", Type: maintenance.WindowTypeManual, Config: maintenance.ManualMaintenanceWindowConfig{Active: active}}
}
func (m ObjectMother) Result() *probe.Result {
	return &probe.Result{ID: uuid.New(), Target: m.Target(), LatencyMs: 100, StatusCode: sql.NullInt32{Int32: 200, Valid: true}, ProbeTime: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), ProcessingStatus: probe.ProcessingStatusNew}
}

// MonitorBuilder changes only the dimensions relevant to a scenario.
// Build never shares mutable fixture data between calls.
type MonitorBuilder struct {
	label    string
	active   bool
	status   monitor.Status
	latency  int
	interval int32
	login    string
}

func NewMonitorBuilder() *MonitorBuilder {
	return &MonitorBuilder{label: "API", active: true, status: monitor.StatusUnknown, latency: 1000, interval: 30, login: "alice"}
}
func (b *MonitorBuilder) WithLabel(v string) *MonitorBuilder          { b.label = v; return b }
func (b *MonitorBuilder) WithActive(v bool) *MonitorBuilder           { b.active = v; return b }
func (b *MonitorBuilder) WithStatus(v monitor.Status) *MonitorBuilder { b.status = v; return b }
func (b *MonitorBuilder) WithMaxLatency(v int) *MonitorBuilder        { b.latency = v; return b }
func (b *MonitorBuilder) WithInterval(v int32) *MonitorBuilder        { b.interval = v; return b }
func (b *MonitorBuilder) WithOwner(v string) *MonitorBuilder          { b.login = v; return b }
func (b *MonitorBuilder) Build() *monitor.Monitor {
	return &monitor.Monitor{ID: uuid.New(), Label: b.label, Target: (ObjectMother{}).Target(), User: &user.User{Login: b.login}, IsActive: b.active, CurrentStatus: b.status, ProbeIntervalSec: b.interval, Expectations: monitor.HTTPExpectations{StatusCodes: []int{200}, MaxLatencyMs: b.latency}, CreatedAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
}
