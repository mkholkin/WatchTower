package testsupport

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// Case keeps Lab 2 results separate from the existing Lab 1 suite.
func Case(t *testing.T, suite, id string, body func(*testing.T, *allure.Context)) {
	t.Helper()
	allure.Wrap(t, func(a *allure.Context) {
		a.Step("Сценарий и очистка данных", func(*allure.Context) {
			t.Run("body", func(child *testing.T) { body(child, a) })
		})
	}, allure.WithParentSuite("WatchTower — ЛР №2"), allure.WithSuite(suite),
		allure.WithTestCaseID("lab2/"+suite+"/"+id), allure.WithTag("lab2"))
}

func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	if os.Getenv("LAB2_TEST_ENV") != "1" {
		t.Fatal("LAB2_TEST_ENV=1 is required for Lab 2 tests")
	}
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s is required", key)
	}
	return v
}

func Logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// Database truncates only application tables. Migration and extension schemas survive.
func Database(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := RequireEnv(t, "LAB2_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme != "postgres" && u.Scheme != "postgresql" || u.Hostname() != "postgres" || strings.TrimPrefix(u.Path, "/") != "postgres" {
		t.Fatalf("LAB2_DATABASE_URL must target the isolated Compose postgres service and postgres database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	ResetDatabase(t, pool)
	t.Cleanup(func() { ResetDatabase(t, pool) })
	return pool
}

func ResetDatabase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, `TRUNCATE TABLE
		"monitor_alert_contact", "maintenance_window_monitor", "monitor_status_log",
		"probe_result", "monitor", "alert_contact", "maintenance_window", "target", "user"
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Errorf("reset Lab 2 application data: %v", err)
	}
}

func Redis(t *testing.T) *goredis.Client {
	t.Helper()
	raw := RequireEnv(t, "LAB2_REDIS_URL")
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() != "redis" || u.Path != "/0" {
		t.Fatal("LAB2_REDIS_URL must target isolated Compose redis DB 0")
	}
	opts, err := goredis.ParseURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	client := goredis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis: %v", err)
		}
	})
	flush := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Errorf("flush Lab 2 Redis: %v", err)
		}
	}
	flush()
	t.Cleanup(flush)
	return client
}
