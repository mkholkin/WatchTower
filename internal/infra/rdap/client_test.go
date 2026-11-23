package rdap_test

import (
	"WatchTower/internal/infra/rdap"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLookupRegistration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/com/v1/domain/example.com" || r.Header.Get("Accept") != "application/rdap+json" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/rdap+json")
		w.Write([]byte(`{"objectClassName":"domain","ldhName":"EXAMPLE.COM","events":[{"eventAction":"expiration","eventDate":"2027-08-13T04:00:00Z"},{"eventAction":"registration","eventDate":"1995-08-14T04:00:00Z"}],"extra":true}`))
	}))
	defer server.Close()
	client, err := rdap.New(server.URL+"/com/v1/", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Lookup(context.Background(), "https://probe.Example.com:8081/health")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "example.com" || got.RegisteredAt == nil || got.ExpiresAt == nil || got.ExpiresAt.Year() != 2027 || got.SourceURL != server.URL+"/com/v1/domain/example.com" {
		t.Fatalf("unexpected registration %+v", got)
	}
}

func TestLookupUpstreamFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"notfound", 404, `{"errorCode":404}`, rdap.ErrNotFound},
		{"limited", 429, `{"errorCode":429}`, rdap.ErrRateLimited},
		{"failure", 500, `{}`, rdap.ErrUnavailable},
		{"redirect", 302, `{}`, rdap.ErrUnavailable},
		{"brokenjson", 200, `{`, rdap.ErrUnavailable},
		{"extraJSON", 200, `{"objectClassName":"domain","ldhName":"example.com"}{}`, rdap.ErrUnavailable},
		{"wrongclass", 200, `{"objectClassName":"entity","ldhName":"example.com"}`, rdap.ErrUnavailable},
		{"wrongdomain", 200, `{"objectClassName":"domain","ldhName":"other.com"}`, rdap.ErrUnavailable},
		{"baddate", 200, `{"objectClassName":"domain","ldhName":"example.com","events":[{"eventAction":"expiration","eventDate":"bad"}]}`, rdap.ErrUnavailable},
		{"large", 200, strings.Repeat(" ", 1048577), rdap.ErrUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "http://127.0.0.1:1")
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, _ := rdap.New(server.URL, time.Second)
			_, err := client.Lookup(context.Background(), "http://example.com")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v; want %v", err, tc.want)
			}
		})
	}
}

func TestLookupRejectsInvalidEndpoints(t *testing.T) {
	client, _ := rdap.New("http://127.0.0.1:1", time.Second)
	for _, endpoint := range []string{"http://127.0.0.1", "http://example.net", "example.com", "ftp://example.com", "https://user@example.com", "https://bad_.com", "https://-bad.com", "https://a..example.com", "https://é.com", "https://com"} {
		_, err := client.Lookup(context.Background(), endpoint)
		if !errors.Is(err, rdap.ErrInvalidDomain) {
			t.Errorf("%s: %v", endpoint, err)
		}
	}
}

func TestLookupMissingDates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"objectClassName":"domain","ldhName":"example.com"}`))
	}))
	defer server.Close()
	client, _ := rdap.New(server.URL, time.Second)
	got, err := client.Lookup(context.Background(), "http://example.com.")
	if err != nil || got.RegisteredAt != nil || got.ExpiresAt != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestLookupTimeoutAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client, _ := rdap.New(server.URL, 20*time.Millisecond)
	_, err := client.Lookup(context.Background(), "https://example.com")
	if !errors.Is(err, rdap.ErrTimeout) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Lookup(ctx, "https://example.com")
	if !errors.Is(err, rdap.ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	for _, base := range []string{"", "://", "ftp://example.com", "https://u:p@example.com", "https://example.com?q=x", "https://example.com#x"} {
		if _, err := rdap.New(base, time.Second); err == nil {
			t.Errorf("accepted %q", base)
		}
	}
	if _, err := rdap.New("https://example.com", 0); err == nil {
		t.Fatal("accepted zero timeout")
	}
}

func TestLookupConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := server.URL
	server.Close()
	client, _ := rdap.New(base, time.Second)
	_, err := client.Lookup(context.Background(), "https://example.com")
	if !errors.Is(err, rdap.ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestLookupDoesNotFollowRedirects(t *testing.T) {
	followed := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true; w.WriteHeader(500) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer source.Close()
	client, _ := rdap.New(source.URL, time.Second)
	_, err := client.Lookup(context.Background(), "https://example.com")
	if !errors.Is(err, rdap.ErrUnavailable) || followed {
		t.Fatalf("error=%v followed=%v", err, followed)
	}
}
