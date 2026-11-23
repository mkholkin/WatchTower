package rdap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidDomain = errors.New("RDAP supports valid .com HTTP(S) monitor endpoints")
	ErrNotFound      = errors.New("domain registration not found")
	ErrUnavailable   = errors.New("registration provider unavailable")
	ErrTimeout       = errors.New("registration provider timed out")
	ErrRateLimited   = errors.New("registration provider rate limited")
	labelPattern     = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
)

const maxResponseBytes = 1 << 20

type Registration struct {
	Domain       string     `json:"domain"`
	RegisteredAt *time.Time `json:"registered_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	SourceURL    string     `json:"source_url"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, ErrUnavailable
	}
	if !validBase(base) || timeout <= 0 {
		return nil, ErrUnavailable
	}
	return &Client{baseURL: strings.TrimRight(base.String(), "/"), http: &http.Client{Timeout: timeout, CheckRedirect: rejectRedirect}}, nil
}

func validBase(base *url.URL) bool {
	return (base.Scheme == "http" || base.Scheme == "https") && base.Hostname() != "" && base.User == nil && base.RawQuery == "" && !base.ForceQuery && base.Fragment == ""
}

func rejectRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func monitorDomain(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", ErrInvalidDomain
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrInvalidDomain
	}
	if parsed.User != nil {
		return "", ErrInvalidDomain
	}
	return registrableDomain(parsed.Hostname())
}

func registrableDomain(host string) (string, error) {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if len(host) > 253 || net.ParseIP(host) != nil {
		return "", ErrInvalidDomain
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 || labels[len(labels)-1] != "com" {
		return "", ErrInvalidDomain
	}
	for _, label := range labels {
		if !labelPattern.MatchString(label) {
			return "", ErrInvalidDomain
		}
	}
	return strings.Join(labels[len(labels)-2:], "."), nil
}

func (c *Client) Lookup(ctx context.Context, endpoint string) (Registration, error) {
	domain, err := monitorDomain(endpoint)
	if err != nil {
		return Registration{}, err
	}
	source := c.baseURL + "/domain/" + domain
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return Registration{}, ErrUnavailable
	}
	request.Header.Set("Accept", "application/rdap+json")
	response, err := c.http.Do(request)
	if err != nil {
		return Registration{}, requestError(err)
	}
	defer response.Body.Close()
	return readResponse(response, domain, source)
}

func requestError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTimeout
	}
	return ErrUnavailable
}

func readResponse(response *http.Response, domain, source string) (Registration, error) {
	if err := responseError(response.StatusCode); err != nil {
		return Registration{}, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return Registration{}, requestError(err)
	}
	if len(body) > maxResponseBytes {
		return Registration{}, ErrUnavailable
	}
	return decodeRegistration(body, domain, source)
}

func responseError(status int) error {
	switch status {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return ErrUnavailable
	}
}

type vendorEvent struct {
	Action string    `json:"eventAction"`
	Date   time.Time `json:"eventDate"`
}
type vendorRegistration struct {
	Class  string        `json:"objectClassName"`
	Name   string        `json:"ldhName"`
	Events []vendorEvent `json:"events"`
}

func decodeRegistration(body []byte, domain, source string) (Registration, error) {
	var record vendorRegistration
	if err := json.Unmarshal(body, &record); err != nil {
		return Registration{}, ErrUnavailable
	}
	if record.Class != "domain" || !strings.EqualFold(strings.TrimSuffix(record.Name, "."), domain) {
		return Registration{}, ErrUnavailable
	}
	result := Registration{Domain: domain, SourceURL: source}
	for _, event := range record.Events {
		if err := assignEvent(&result, event); err != nil {
			return Registration{}, err
		}
	}
	return result, nil
}

func assignEvent(result *Registration, event vendorEvent) error {
	if event.Date.IsZero() {
		return ErrUnavailable
	}
	switch event.Action {
	case "registration":
		result.RegisteredAt = &event.Date
	case "expiration":
		result.ExpiresAt = &event.Date
	}
	return nil
}
