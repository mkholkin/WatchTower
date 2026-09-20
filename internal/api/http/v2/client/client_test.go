package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientLogsInAndUsesBearerTokenForMonitorList(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status := http.StatusOK
		body := ""
		switch request.URL.Path {
		case "/api/v2/auth/tokens":
			if request.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", request.Method)
			}
			body = `{"access_token":"signed-token","token_type":"Bearer"}`
		case "/api/v2/monitors":
			if request.Header.Get("Authorization") != "Bearer signed-token" {
				t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
			}
			body = `[{"id":"1b6cae53-a014-4b79-843c-6433ad451b4e","label":"Public API","endpoint":"https://api.example.com/health","protocol":"HTTP","probe_interval_sec":30,"network_config":{"method":"GET","follow_redirects":true},"expectations":{"expected_status_codes":[200],"max_response_time_ms":1500},"is_enabled":true,"status":"up","alert_contact_ids":[],"maintenance_window_ids":[],"created_at":"2026-09-20T10:00:00Z"}]`
		default:
			status = http.StatusNotFound
			body = `{"code":"not_found","message":"not found"}`
		}
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}

	client, err := New("http://watchtower.test", apiv2gen.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	token, err := client.Login(context.Background(), "alice", "correct-horse")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "signed-token" {
		t.Fatalf("Login() token = %q", token)
	}

	monitors, err := client.GetAllMonitors(context.Background())
	if err != nil {
		t.Fatalf("GetAllMonitors() error = %v", err)
	}
	if len(monitors) != 1 || monitors[0].Label != "Public API" || monitors[0].Target.Endpoint != "https://api.example.com/health" {
		t.Fatalf("GetAllMonitors() = %#v", monitors)
	}
}
