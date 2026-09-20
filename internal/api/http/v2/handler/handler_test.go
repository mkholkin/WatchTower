package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	authsvc "WatchTower/internal/service/auth"
	contactssvc "WatchTower/internal/service/contacts"
	contactdto "WatchTower/internal/service/contacts/dto"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitorsvc "WatchTower/internal/service/monitoring_management"
	monitordto "WatchTower/internal/service/monitoring_management/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type authServiceStub struct {
	loginToken     string
	loginErr       error
	registerCalled bool
}

func (s *authServiceStub) Register(context.Context, string, string) error {
	s.registerCalled = true
	return nil
}
func (s *authServiceStub) Login(context.Context, string, string) (string, error) {
	return s.loginToken, s.loginErr
}
func (s *authServiceStub) ParseToken(string) (string, error) { return "alice", nil }

type monitoringServiceStub struct {
	monitor      *monitor.Monitor
	getErr       error
	lastUpdate   monitordto.UpdateMonitorDTO
	updateCalled bool
	createCalled bool
}

type contactServiceStub struct {
	createdWith contactdto.CreateTelegramAlertContactDTO
	contact     *alert.Contact
}

type maintenanceServiceStub struct {
	window        *maintenance.MaintenanceWindow
	linkedMonitor []uuid.UUID
	lastUpdate    maintenancesvc.UpdateMaintenanceWindowDTO
	updateCalled  bool
}

type metricsServiceStub struct {
	lastLimit int
	summary   *probe.Summary
}

func (s *metricsServiceStub) GetLastSummaries(_ context.Context, _ uuid.UUID, limit int) ([]*probe.Summary, error) {
	s.lastLimit = limit
	return []*probe.Summary{s.summary}, nil
}
func (s *metricsServiceStub) GetSummariesForPeriod(context.Context, uuid.UUID, time.Time, time.Time) ([]*probe.Summary, error) {
	return []*probe.Summary{s.summary}, nil
}
func (s *metricsServiceStub) GetSLA(_ context.Context, monitorID uuid.UUID, from, to time.Time) (metricssvc.SLAStats, error) {
	return metricssvc.SLAStats{MonitorID: monitorID, PeriodStart: from, PeriodEnd: to}, nil
}
func (s *metricsServiceStub) GetStatusHistory(context.Context, uuid.UUID, time.Time, time.Time) ([]metricssvc.StatusEvent, error) {
	return nil, nil
}

func (s *maintenanceServiceStub) GetAllMaintenanceWindows(context.Context) ([]maintenance.MaintenanceWindow, error) {
	return []maintenance.MaintenanceWindow{*s.window}, nil
}
func (s *maintenanceServiceStub) GetMaintenanceWindow(context.Context, uuid.UUID) (*maintenance.MaintenanceWindow, error) {
	return s.window, nil
}
func (s *maintenanceServiceStub) CreateOneTimeMaintenanceWindow(_ context.Context, dto maintenancesvc.CreateOneTimeMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error) {
	s.window = &maintenance.MaintenanceWindow{
		ID:          uuid.MustParse("de8f8207-058b-43d5-8466-a0216c7b57fb"),
		Title:       dto.Title,
		Description: dto.Description,
		Type:        maintenance.WindowTypeOneTime,
		Config: maintenance.OneTimeMaintenanceWindowConfig{
			StartTime: dto.StartTime,
			EndTime:   dto.EndTime,
		},
	}
	return s.window, nil
}
func (s *maintenanceServiceStub) CreateManualMaintenanceWindow(_ context.Context, dto maintenancesvc.CreateManualMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error) {
	s.window = &maintenance.MaintenanceWindow{ID: uuid.New(), Title: dto.Title, Description: dto.Description, Type: maintenance.WindowTypeManual, Config: maintenance.ManualMaintenanceWindowConfig{Active: dto.Active}}
	return s.window, nil
}
func (s *maintenanceServiceStub) AddMonitorToMaintenanceWindow(_ context.Context, monitorID uuid.UUID, _ uuid.UUID) error {
	s.linkedMonitor = append(s.linkedMonitor, monitorID)
	return nil
}
func (s *maintenanceServiceStub) RemoveMonitorFromMaintenanceWindow(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (s *maintenanceServiceStub) UpdateMaintenanceWindow(_ context.Context, dto maintenancesvc.UpdateMaintenanceWindowDTO) error {
	s.lastUpdate = dto
	s.updateCalled = true
	if update, ok := dto.ConfigUpdate.(maintenance.ManualConfigUpdate); ok && update.Active != nil {
		s.window.Config = maintenance.ManualMaintenanceWindowConfig{Active: *update.Active}
	}
	return nil
}
func (s *maintenanceServiceStub) DeleteMaintenanceWindow(context.Context, uuid.UUID) error {
	return nil
}

func (s *contactServiceStub) CreateTelegramAlertContact(_ context.Context, dto contactdto.CreateTelegramAlertContactDTO) (*alert.Contact, error) {
	s.createdWith = dto
	return s.contact, nil
}
func (s *contactServiceStub) GetAllAlertContacts(context.Context) ([]alert.Contact, error) {
	return []alert.Contact{*s.contact}, nil
}
func (s *contactServiceStub) GetAlertContact(context.Context, uuid.UUID) (*alert.Contact, error) {
	return s.contact, nil
}
func (s *contactServiceStub) UpdateAlertContact(context.Context, contactdto.UpdateAlertContactDTO) error {
	return nil
}
func (s *contactServiceStub) DeleteAlertContact(context.Context, uuid.UUID) error { return nil }
func (s *contactServiceStub) EnableAlertContact(context.Context, uuid.UUID) error { return nil }
func (s *contactServiceStub) DisableAlertContact(context.Context, uuid.UUID) error {
	return nil
}

func (s *monitoringServiceStub) GetAllMonitors(context.Context) ([]*monitor.Monitor, error) {
	return []*monitor.Monitor{s.monitor}, nil
}
func (s *monitoringServiceStub) GetMonitor(context.Context, uuid.UUID) (*monitor.Monitor, error) {
	return s.monitor, s.getErr
}
func (s *monitoringServiceStub) DisableMonitor(context.Context, uuid.UUID) error { return nil }
func (s *monitoringServiceStub) EnableMonitor(context.Context, uuid.UUID) error  { return nil }
func (s *monitoringServiceStub) DeleteMonitor(context.Context, uuid.UUID) error  { return nil }
func (s *monitoringServiceStub) CreateMonitor(context.Context, monitordto.CreateMonitorDTO) (*monitor.Monitor, error) {
	s.createCalled = true
	return s.monitor, nil
}
func (s *monitoringServiceStub) UpdateMonitor(_ context.Context, update monitordto.UpdateMonitorDTO) error {
	s.lastUpdate = update
	s.updateCalled = true
	if update.Label != nil {
		s.monitor.Label = *update.Label
	}
	if update.ProbeIntervalSec != nil {
		s.monitor.ProbeIntervalSec = *update.ProbeIntervalSec
	}
	return nil
}
func (s *monitoringServiceStub) LinkAlertContact(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (s *monitoringServiceStub) UnlinkAlertContact(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func testMonitor() *monitor.Monitor {
	id := uuid.MustParse("1b6cae53-a014-4b79-843c-6433ad451b4e")
	return &monitor.Monitor{
		ID:               id,
		Label:            "Public API",
		Target:           &target.Target{Endpoint: "https://api.example.com/health", Config: target.HTTPConfig{Method: "GET", FollowRedirects: true}},
		CurrentStatus:    monitor.StatusUp,
		ProbeIntervalSec: 30,
		IsActive:         true,
		CreatedAt:        time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		Expectations:     monitor.HTTPExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1500},
	}
}

func TestLoginUserReturnsBearerToken(t *testing.T) {
	h := NewApiHandler(&authServiceStub{loginToken: "signed-token"}, nil, nil, nil, nil)

	response, err := h.LoginUser(context.Background(), apiv2gen.LoginUserRequestObject{
		Body: &apiv2gen.LoginUserJSONRequestBody{Login: "alice", Password: "correct-horse"},
	})
	if err != nil {
		t.Fatalf("LoginUser() error = %v", err)
	}

	got, ok := response.(apiv2gen.LoginUser200JSONResponse)
	if !ok {
		t.Fatalf("LoginUser() response = %T, want LoginUser200JSONResponse", response)
	}
	if got.AccessToken != "signed-token" || got.TokenType != apiv2gen.Bearer {
		t.Fatalf("LoginUser() = %#v, want bearer signed-token", got)
	}
}

func TestRegisterUserRejectsCredentialsOutsideContract(t *testing.T) {
	auth := &authServiceStub{}
	h := NewApiHandler(auth, nil, nil, nil, nil)

	response, err := h.RegisterUser(context.Background(), apiv2gen.RegisterUserRequestObject{
		Body: &apiv2gen.RegisterUserJSONRequestBody{Login: "ab", Password: "short"},
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if _, ok := response.(apiv2gen.RegisterUser400JSONResponse); !ok {
		t.Fatalf("RegisterUser() response = %T, want RegisterUser400JSONResponse", response)
	}
	if auth.registerCalled {
		t.Fatal("RegisterUser() called service for credentials rejected by the contract")
	}
}

func TestGetMonitorMapsRepositoryNotFoundTo404(t *testing.T) {
	monitoring := &monitoringServiceStub{getErr: repo.ErrNotFound}
	h := NewApiHandler(nil, monitoring, nil, nil, nil)

	response, err := h.GetMonitor(context.Background(), apiv2gen.GetMonitorRequestObject{MonitorId: uuid.New()})
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
	if _, ok := response.(apiv2gen.GetMonitor404JSONResponse); !ok {
		t.Fatalf("GetMonitor() response = %T, want GetMonitor404JSONResponse", response)
	}
}

func TestCreateMonitorRejectsInvalidEndpointBeforeService(t *testing.T) {
	monitoring := &monitoringServiceStub{monitor: testMonitor()}
	h := NewApiHandler(nil, monitoring, nil, nil, nil)

	response, err := h.CreateMonitor(context.Background(), apiv2gen.CreateMonitorRequestObject{
		Body: &apiv2gen.CreateMonitorJSONRequestBody{
			Label: "Broken", Endpoint: "not a URL", ProbeIntervalSec: 30,
			NetworkConfig: apiv2gen.HTTPConfig{Method: apiv2gen.GET, FollowRedirects: true},
			Expectations:  apiv2gen.HTTPExpectations{ExpectedStatusCodes: []int{200}, MaxResponseTimeMs: 1000},
		},
	})
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}
	if _, ok := response.(apiv2gen.CreateMonitor400JSONResponse); !ok {
		t.Fatalf("CreateMonitor() response = %T, want CreateMonitor400JSONResponse", response)
	}
	if monitoring.createCalled {
		t.Fatal("CreateMonitor() called service for endpoint rejected by the contract")
	}
}

func TestPatchMonitorSendsOnlyProvidedFieldsAndReturnsUpdatedResource(t *testing.T) {
	monitoring := &monitoringServiceStub{monitor: testMonitor()}
	h := NewApiHandler(nil, monitoring, nil, nil, nil)
	newLabel := "Public API v2"

	response, err := h.PatchMonitor(context.Background(), apiv2gen.PatchMonitorRequestObject{
		MonitorId: monitoring.monitor.ID,
		Body:      &apiv2gen.PatchMonitorJSONRequestBody{Label: &newLabel},
	})
	if err != nil {
		t.Fatalf("PatchMonitor() error = %v", err)
	}
	if !monitoring.updateCalled {
		t.Fatal("PatchMonitor() did not call UpdateMonitor")
	}
	if monitoring.lastUpdate.Label == nil || *monitoring.lastUpdate.Label != newLabel {
		t.Fatalf("UpdateMonitor label = %v, want %q", monitoring.lastUpdate.Label, newLabel)
	}
	if monitoring.lastUpdate.Endpoint != nil || monitoring.lastUpdate.ProbeIntervalSec != nil {
		t.Fatalf("UpdateMonitor received fields absent from PATCH: %#v", monitoring.lastUpdate)
	}

	got, ok := response.(apiv2gen.PatchMonitor200JSONResponse)
	if !ok {
		t.Fatalf("PatchMonitor() response = %T, want PatchMonitor200JSONResponse", response)
	}
	if got.Label != newLabel {
		t.Fatalf("PatchMonitor() label = %q, want %q", got.Label, newLabel)
	}
}

func TestCreateAlertContactMapsTelegramCredentialsAndLocation(t *testing.T) {
	contactID := uuid.MustParse("8f73085c-82cf-4c6e-9724-5f3f85ecb65e")
	contacts := &contactServiceStub{contact: &alert.Contact{
		ID:       contactID,
		Name:     "Operations chat",
		Type:     alert.ContactTypeTelegram,
		Config:   alert.TelegramContactConfig{ChatID: 123456789, BotToken: "secret"},
		IsActive: true,
	}}
	h := NewApiHandler(nil, nil, contacts, nil, nil)
	token := "123456:example-token"

	response, err := h.CreateAlertContact(context.Background(), apiv2gen.CreateAlertContactRequestObject{
		Body: &apiv2gen.CreateAlertContactJSONRequestBody{
			Name:     "Operations chat",
			Platform: apiv2gen.AlertContactCreatePlatformTelegram,
			Credentials: apiv2gen.TelegramCredentials{
				ChatId:   123456789,
				BotToken: &token,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertContact() error = %v", err)
	}
	if contacts.createdWith.BotToken != token || contacts.createdWith.ChatID != 123456789 {
		t.Fatalf("CreateAlertContact() dto = %#v", contacts.createdWith)
	}

	got, ok := response.(apiv2gen.CreateAlertContact201JSONResponse)
	if !ok {
		t.Fatalf("CreateAlertContact() response = %T, want CreateAlertContact201JSONResponse", response)
	}
	if got.Body.Id != contactID || got.Headers.Location == nil || *got.Headers.Location != "/api/v2/alert-contacts/"+contactID.String() {
		t.Fatalf("CreateAlertContact() response = %#v", got)
	}
}

func TestCreateAlertContactRejectsMissingBotToken(t *testing.T) {
	contacts := &contactServiceStub{}
	h := NewApiHandler(nil, nil, contacts, nil, nil)

	response, err := h.CreateAlertContact(context.Background(), apiv2gen.CreateAlertContactRequestObject{
		Body: &apiv2gen.CreateAlertContactJSONRequestBody{
			Name:        "Operations chat",
			Platform:    apiv2gen.AlertContactCreatePlatformTelegram,
			Credentials: apiv2gen.TelegramCredentials{ChatId: 123456789},
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertContact() error = %v", err)
	}
	if _, ok := response.(apiv2gen.CreateAlertContact400JSONResponse); !ok {
		t.Fatalf("CreateAlertContact() response = %T, want CreateAlertContact400JSONResponse", response)
	}
}

func TestCreateAlertContactRejectsNameOutsideContract(t *testing.T) {
	contactService := &contactServiceStub{contact: &alert.Contact{ID: uuid.New(), Name: "unused", Config: alert.TelegramContactConfig{}}}
	h := NewApiHandler(nil, nil, contactService, nil, nil)
	token := "secret"

	response, err := h.CreateAlertContact(context.Background(), apiv2gen.CreateAlertContactRequestObject{
		Body: &apiv2gen.CreateAlertContactJSONRequestBody{
			Name: strings.Repeat("я", 121), Platform: apiv2gen.AlertContactCreatePlatformTelegram,
			Credentials: apiv2gen.TelegramCredentials{ChatId: 123, BotToken: &token},
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertContact() error = %v", err)
	}
	if _, ok := response.(apiv2gen.CreateAlertContact400JSONResponse); !ok {
		t.Fatalf("CreateAlertContact() response = %T, want CreateAlertContact400JSONResponse", response)
	}
}

func TestCreateMonitorRejectsDuplicateRelations(t *testing.T) {
	monitorID := uuid.New()
	service := &monitoringServiceStub{}
	h := NewApiHandler(nil, service, nil, nil, nil)

	response, err := h.CreateMonitor(context.Background(), apiv2gen.CreateMonitorRequestObject{
		Body: &apiv2gen.CreateMonitorJSONRequestBody{
			Label: "API", Endpoint: "https://example.com", ProbeIntervalSec: 30,
			NetworkConfig:   apiv2gen.HTTPConfig{Method: apiv2gen.GET, FollowRedirects: true},
			Expectations:    apiv2gen.HTTPExpectations{ExpectedStatusCodes: []int{200}, MaxResponseTimeMs: 1000},
			AlertContactIds: &[]apiv2gen.UUID{monitorID, monitorID},
		},
	})
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}
	if _, ok := response.(apiv2gen.CreateMonitor400JSONResponse); !ok {
		t.Fatalf("CreateMonitor() response = %T, want CreateMonitor400JSONResponse", response)
	}
}

func TestCreateMaintenanceWindowCreatesAndLinksMonitors(t *testing.T) {
	maintenanceService := &maintenanceServiceStub{}
	monitorID := uuid.MustParse("1b6cae53-a014-4b79-843c-6433ad451b4e")
	monitoringService := &monitoringServiceStub{monitor: &monitor.Monitor{ID: monitorID}}
	h := NewApiHandler(nil, monitoringService, nil, maintenanceService, nil)
	startsAt := time.Date(2026, 9, 21, 1, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(time.Hour)
	var config apiv2gen.MaintenanceConfig
	if err := config.FromOneTimeMaintenanceConfig(apiv2gen.OneTimeMaintenanceConfig{
		Type: apiv2gen.OneTime, StartsAt: startsAt, EndsAt: endsAt,
	}); err != nil {
		t.Fatalf("build maintenance config: %v", err)
	}
	monitorIDs := []apiv2gen.UUID{monitorID}
	description := "Planned PostgreSQL maintenance"

	response, err := h.CreateMaintenanceWindow(context.Background(), apiv2gen.CreateMaintenanceWindowRequestObject{
		Body: &apiv2gen.CreateMaintenanceWindowJSONRequestBody{
			Title: "Database migration", Description: &description, Config: config, MonitorIds: &monitorIDs,
		},
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceWindow() error = %v", err)
	}
	if len(maintenanceService.linkedMonitor) != 1 || maintenanceService.linkedMonitor[0] != monitorID {
		t.Fatalf("linked monitors = %v, want [%s]", maintenanceService.linkedMonitor, monitorID)
	}

	got, ok := response.(apiv2gen.CreateMaintenanceWindow201JSONResponse)
	if !ok {
		t.Fatalf("CreateMaintenanceWindow() response = %T, want CreateMaintenanceWindow201JSONResponse", response)
	}
	if len(got.Body.MonitorIds) != 1 || got.Body.MonitorIds[0] != monitorID {
		t.Fatalf("response monitor_ids = %v, want [%s]", got.Body.MonitorIds, monitorID)
	}
}

func TestCreateManualMaintenanceWindowHonorsActiveFlag(t *testing.T) {
	maintenanceService := &maintenanceServiceStub{}
	h := NewApiHandler(nil, nil, nil, maintenanceService, nil)
	var config apiv2gen.MaintenanceConfig
	if err := config.FromManualMaintenanceConfig(apiv2gen.ManualMaintenanceConfig{
		Type: apiv2gen.Manual, IsActive: true,
	}); err != nil {
		t.Fatalf("build maintenance config: %v", err)
	}

	response, err := h.CreateMaintenanceWindow(context.Background(), apiv2gen.CreateMaintenanceWindowRequestObject{
		Body: &apiv2gen.CreateMaintenanceWindowJSONRequestBody{Title: "Emergency work", Config: config},
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceWindow() error = %v", err)
	}
	if maintenanceService.updateCalled {
		t.Fatal("CreateMaintenanceWindow() performed a second, non-atomic update")
	}
	created, ok := response.(apiv2gen.CreateMaintenanceWindow201JSONResponse)
	if !ok {
		t.Fatalf("CreateMaintenanceWindow() response = %T, want CreateMaintenanceWindow201JSONResponse", response)
	}
	manual, err := created.Body.Config.AsManualMaintenanceConfig()
	if err != nil {
		t.Fatalf("read response config: %v", err)
	}
	if !manual.IsActive {
		t.Fatal("response config is_active = false, want true")
	}
}

func TestCreateMaintenanceWindowRejectsConfigOutsideContract(t *testing.T) {
	for name, rawConfig := range map[string]string{
		"unknown property":  `{"type":"manual","is_active":false,"unknown":true}`,
		"missing is_active": `{"type":"manual"}`,
	} {
		t.Run(name, func(t *testing.T) {
			maintenanceService := &maintenanceServiceStub{}
			h := NewApiHandler(nil, nil, nil, maintenanceService, nil)
			var config apiv2gen.MaintenanceConfig
			if err := json.Unmarshal([]byte(rawConfig), &config); err != nil {
				t.Fatalf("decode test config: %v", err)
			}

			response, err := h.CreateMaintenanceWindow(context.Background(), apiv2gen.CreateMaintenanceWindowRequestObject{
				Body: &apiv2gen.CreateMaintenanceWindowJSONRequestBody{Title: "Maintenance", Config: config},
			})
			if err != nil {
				t.Fatalf("CreateMaintenanceWindow() error = %v", err)
			}
			if _, ok := response.(apiv2gen.CreateMaintenanceWindow400JSONResponse); !ok {
				t.Fatalf("CreateMaintenanceWindow() response = %T, want CreateMaintenanceWindow400JSONResponse", response)
			}
		})
	}
}

func TestListMonitorChecksUsesLimitAndMapsSummary(t *testing.T) {
	monitorID := uuid.MustParse("1b6cae53-a014-4b79-843c-6433ad451b4e")
	checkedAt := time.Date(2026, 9, 20, 10, 5, 0, 0, time.UTC)
	metrics := &metricsServiceStub{summary: &probe.Summary{
		MonitorID: monitorID, ProbeTime: checkedAt, MonitorStatus: monitor.StatusUp,
		LatencyMs: 84, StatusCode: 200, NetworkFailure: false,
	}}
	h := NewApiHandler(nil, nil, nil, nil, metrics)
	limit := apiv2gen.Limit(25)

	response, err := h.ListMonitorChecks(context.Background(), apiv2gen.ListMonitorChecksRequestObject{
		MonitorId: monitorID,
		Params:    apiv2gen.ListMonitorChecksParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListMonitorChecks() error = %v", err)
	}
	if metrics.lastLimit != 25 {
		t.Fatalf("GetLastSummaries() limit = %d, want 25", metrics.lastLimit)
	}
	got, ok := response.(apiv2gen.ListMonitorChecks200JSONResponse)
	if !ok || len(got) != 1 {
		t.Fatalf("ListMonitorChecks() response = %#v", response)
	}
	if got[0].CheckedAt != checkedAt || got[0].StatusCode == nil || *got[0].StatusCode != 200 {
		t.Fatalf("ListMonitorChecks() item = %#v", got[0])
	}
}

func TestV2HTTPPatchRequiresAuthAndReturnsContractResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := &authServiceStub{}
	monitoring := &monitoringServiceStub{monitor: testMonitor()}
	h := NewApiHandler(auth, monitoring, nil, nil, nil)
	router := gin.New()
	strict := apiv2gen.NewStrictHandler(h, []apiv2gen.StrictMiddlewareFunc{h.AuthStrictMiddleware})
	apiv2gen.RegisterHandlers(router, strict)

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v2/monitors", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized list status = %d, want 401", unauthorized.Code)
	}

	payload := []byte(`{"label":"Public API v2"}`)
	request := httptest.NewRequest(http.MethodPatch, "/api/v2/monitors/"+monitoring.monitor.ID.String(), bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer signed-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var body apiv2gen.Monitor
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode PATCH response: %v", err)
	}
	if body.Label != "Public API v2" {
		t.Fatalf("PATCH label = %q, want Public API v2", body.Label)
	}
}

var _ authsvc.AuthService = (*authServiceStub)(nil)
var _ monitorsvc.MonitoringManagementService = (*monitoringServiceStub)(nil)
var _ contactssvc.ContactService = (*contactServiceStub)(nil)
var _ maintenancesvc.MaintenanceService = (*maintenanceServiceStub)(nil)
var _ metricssvc.MetricQueryService = (*metricsServiceStub)(nil)
