//go:build lab2e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

type apiClient struct {
	base  string
	token string
	http  *http.Client
}

func (c *apiClient) request(t *testing.T, method, path string, body any, want int, result any) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(data)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d, want %d, body %s", method, path, resp.StatusCode, want, data)
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			t.Fatalf("decode %s %s: %v; body %s", method, path, err, data)
		}
	}
}

func (c *apiClient) read(method, path string, result any) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode == 200 && result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func TestHTTPMonitorFromRegistrationToBackgroundCheck(t *testing.T) {
	testsupport.Case(t, "e2e", "http-monitor-lifecycle", func(t *testing.T, a *allure.Context) {
		base := testsupport.RequireEnv(t, "LAB2_API_URL")
		probe := testsupport.RequireEnv(t, "LAB2_PROBE_URL")
		parsed, err := url.Parse(probe)
		if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "probe" {
			t.Fatal("LAB2_PROBE_URL must target isolated Compose probe")
		}
		_ = testsupport.Database(t)
		_ = testsupport.Redis(t)
		c := &apiClient{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 5 * time.Second}}
		credentials := map[string]any{"login": "lab2_alice", "password": "Lab2Pass-123!"}
		a.Step("Регистрация, вход и защита списка", func(*allure.Context) {
			c.request(t, "POST", "/auth/register", credentials, 201, nil)
			var login struct {
				AccessToken string `json:"access_token"`
			}
			c.request(t, "POST", "/auth/login", credentials, 200, &login)
			if login.AccessToken == "" {
				t.Fatal("login returned empty token")
			}
			c.request(t, "GET", "/monitors", nil, 401, nil)
			c.token = login.AccessToken
		})
		var id string
		a.Step("Создание реального HTTP-монитора", func(*allure.Context) {
			body := map[string]any{
				"label": "Lab2 probe", "endpoint": probe, "probe_interval": 1,
				"alert_contact_ids": []string{}, "maintenance_window_ids": []string{},
				"network_config": map[string]any{"protocol": "HTTP", "method": "GET", "headers": map[string]string{}, "body": "", "follow_redirects": false},
				"expectations":   map[string]any{"protocol": "HTTP", "expected_status_codes": []int{200}, "expected_response_time_ms": 5000},
			}
			var created struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Label  string `json:"label"`
			}
			c.request(t, "POST", "/monitors", body, 201, &created)
			if created.ID == "" || created.Label != "Lab2 probe" {
				t.Fatalf("created monitor: %#v", created)
			}
			id = created.ID
		})
		path := "/monitors/" + id
		a.Step("Фоновая проверка доводит статус до up и записывает HTTP 200", func(*allure.Context) {
			deadline := time.Now().Add(30 * time.Second)
			last := ""
			for time.Now().Before(deadline) {
				var detail struct {
					Status string `json:"status"`
				}
				status, err := c.read("GET", path, &detail)
				if err != nil {
					last = err.Error()
				} else if status != 200 {
					last = fmt.Sprintf("detail status %d", status)
				} else if detail.Status != "up" {
					last = "monitor status " + detail.Status
				} else {
					var checks []struct {
						Status     string `json:"status"`
						StatusCode *int   `json:"status_code"`
					}
					status, err = c.read("GET", path+"/checks?limit=10", &checks)
					if err != nil {
						last = err.Error()
					} else if status != 200 {
						last = fmt.Sprintf("checks status %d", status)
					} else {
						for _, check := range checks {
							if check.Status == "up" && check.StatusCode != nil && *check.StatusCode == 200 {
								return
							}
						}
						last = fmt.Sprintf("checks have no successful HTTP 200: %#v", checks)
					}
				}
				select {
				case <-time.After(250 * time.Millisecond):
				case <-time.After(time.Until(deadline)):
				}
			}
			t.Fatalf("background probe did not become UP within 30s: %s", last)
		})
		a.Step("Список, детали и SLA показывают монитор", func(*allure.Context) {
			var list []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			c.request(t, "GET", "/monitors", nil, 200, &list)
			if len(list) != 1 || list[0].ID != id || list[0].Status != "up" {
				t.Fatalf("monitors = %#v", list)
			}
			var details struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			c.request(t, "GET", path, nil, 200, &details)
			if details.ID != id || details.Status != "up" {
				t.Fatalf("details = %#v", details)
			}
			var sla struct {
				MonitorID string  `json:"monitor_id"`
				Uptime    float64 `json:"uptime_percentage"`
			}
			c.request(t, "GET", path+"/sla", nil, 200, &sla)
			if sla.MonitorID != id || sla.Uptime < 0 || sla.Uptime > 100 {
				t.Fatalf("SLA = %#v", sla)
			}
		})
		a.Step("Редактирование, отключение, включение и удаление", func(*allure.Context) {
			var updated struct {
				Label string `json:"label"`
			}
			c.request(t, "PATCH", path, map[string]any{"label": "Lab2 probe renamed"}, 200, &updated)
			if updated.Label != "Lab2 probe renamed" {
				t.Fatalf("updated label = %q", updated.Label)
			}
			c.request(t, "POST", path+"/disable", nil, 200, nil)
			var disabled struct {
				IsEnabled bool `json:"is_enabled"`
			}
			c.request(t, "GET", path, nil, 200, &disabled)
			if disabled.IsEnabled {
				t.Fatal("monitor remained enabled")
			}
			c.request(t, "POST", path+"/enable", nil, 200, nil)
			var enabled struct {
				IsEnabled bool `json:"is_enabled"`
			}
			c.request(t, "GET", path, nil, 200, &enabled)
			if !enabled.IsEnabled {
				t.Fatal("monitor remained disabled")
			}
			c.request(t, "DELETE", path, nil, 204, nil)
			c.request(t, "GET", path, nil, 404, nil)
		})
	})
}
