package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	authsvc "WatchTower/internal/service/auth"
	contactssvc "WatchTower/internal/service/contacts"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitorsvc "WatchTower/internal/service/monitoring_management"
)

var (
	_ authsvc.AuthService                    = (*Client)(nil)
	_ monitorsvc.MonitoringManagementService = (*Client)(nil)
	_ contactssvc.ContactService             = (*Client)(nil)
	_ maintenancesvc.MaintenanceService      = (*Client)(nil)
	_ metricssvc.MetricQueryService          = (*Client)(nil)
)

type Client struct {
	api   *apiv2gen.ClientWithResponses
	token string
	login string
}

func New(baseURL string, options ...apiv2gen.ClientOption) (*Client, error) {
	apiClient, err := apiv2gen.NewClientWithResponses(strings.TrimRight(baseURL, "/"), options...)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}
	return &Client{api: apiClient}, nil
}

func (c *Client) withAuth(_ context.Context, request *http.Request) error {
	if c.token == "" {
		return nil
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	return nil
}
