package main

import (
	"WatchTower/configs"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/target"
	notifinfra "WatchTower/internal/infra/notification"
	infra "WatchTower/internal/infra/probe"
	"WatchTower/internal/infra/repository/postgres"
	"WatchTower/internal/infra/repository/redis"
	analyzationsvc "WatchTower/internal/service/analyze"
	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/common/provider"
	contactssvc "WatchTower/internal/service/contacts"
	contact_dto "WatchTower/internal/service/contacts/dto"
	healthchecksvc "WatchTower/internal/service/healthcheck"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitoringsvc "WatchTower/internal/service/monitoring_management"
	"WatchTower/internal/service/monitoring_management/dto"
	notificationsvc "WatchTower/internal/service/notification"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func newPgPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <path-to-config.yaml>\n", os.Args[0])
		return
	}

	configPath := os.Args[1]

	cfg, err := configs.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config %q: %v\n", configPath, err)
		return
	}

	logger, logCloser, err := configs.NewLoggerFromConfig(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		return
	}
	if logCloser != nil {
		defer func() {
			_ = logCloser.Close()
		}()
	}

	authPool, err := newPgPool(ctx, cfg.Database.Auth.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init auth db pool: %v\n", err)
		return
	}
	defer authPool.Close()
	monitoringPool, err := newPgPool(ctx, cfg.Database.Monitoring.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init monitoring db pool: %v\n", err)
		return
	}
	defer monitoringPool.Close()
	maintenancePool, err := newPgPool(ctx, cfg.Database.Maintenance.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init maintenance db pool: %v\n", err)
		return
	}
	defer maintenancePool.Close()
	contactsPool, err := newPgPool(ctx, cfg.Database.Contacts.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init contacts db pool: %v\n", err)
		return
	}
	defer contactsPool.Close()
	metricsPool, err := newPgPool(ctx, cfg.Database.Metrics.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init metrics db pool: %v\n", err)
		return
	}
	defer metricsPool.Close()
	healthcheckerPool, err := newPgPool(ctx, cfg.Database.Healthchecker.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init healthchecker db pool: %v\n", err)
		return
	}
	defer healthcheckerPool.Close()
	analyzerPool, err := pgxpool.New(ctx, cfg.Database.Analyzer.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init analyzer db pool: %v\n", err)
		return
	}
	defer analyzerPool.Close()

	userRepo := postgres.NewUserRepository(authPool, logger)
	jwtTTL := time.Duration(cfg.Auth.JWTTTLHours) * time.Hour
	authService := authsvc.NewService(userRepo, cfg.Auth.JWTSecret, jwtTTL)

	monitoringMonitorRepo := postgres.NewMonitorRepository(monitoringPool, logger)
	monitoringTargetRepo := postgres.NewTargetRepository(monitoringPool, logger)
	monitoringAlertContactRepo := postgres.NewAlertContactRepository(monitoringPool, logger)
	monitoringMWRepo := postgres.NewMaintenanceWindowRepository(monitoringPool, logger)

	maintenanceMonitorRepo := postgres.NewMonitorRepository(maintenancePool, logger)
	maintenanceMWRepo := postgres.NewMaintenanceWindowRepository(maintenancePool, logger)

	contactsAlertContactRepo := postgres.NewAlertContactRepository(contactsPool, logger)

	metricsMonitorRepo := postgres.NewMonitorRepository(metricsPool, logger)
	metricsRepo := postgres.NewMetricsRepository(metricsPool, logger)

	healthTargetRepo := postgres.NewTargetRepository(healthcheckerPool, logger)
	healthProbeResultRepo := postgres.NewProbeResultRepository(healthcheckerPool, logger)

	analyzerMonitorRepo := postgres.NewMonitorRepository(analyzerPool, logger)
	analyzerProbeResultRepo := postgres.NewProbeResultRepository(analyzerPool, logger)
	analyzerProbeSummaryRepo := postgres.NewProbeSummaryRepository(analyzerPool, logger)

	metricsProbeSummaryRepo := postgres.NewProbeSummaryRepository(metricsPool, logger)

	redisOpts, err := goredis.ParseURL(cfg.Redis.URL)
	if err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "invalid redis url in config: %v\n", err); writeErr != nil {
			return
		}
		return
	}
	redisClient := goredis.NewClient(redisOpts)
	defer func() {
		_ = redisClient.Close()
	}()
	probeSummaryRepo := redis.NewProbeSummaryRepository(
		redisClient,
		metricsProbeSummaryRepo,
		logger,
	)

	eventBus := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            256,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
		PreserveContext:                false,
	},
		newFilteredWatermillLogger(watermill.NewStdLogger(false, false)),
	)

	monitoringService := monitoringsvc.NewMonitoringManagementService(
		monitoringMonitorRepo,
		monitoringTargetRepo,
		monitoringAlertContactRepo,
		monitoringMWRepo,
		provider.NewUserProvider(userRepo),
		eventBus,
		logger,
	)

	maintenanceService := maintenancesvc.NewMaintenanceService(
		maintenanceMWRepo,
		maintenanceMonitorRepo,
		provider.NewUserProvider(userRepo),
		logger,
	)

	contactService := contactssvc.NewContactService(
		contactsAlertContactRepo,
		provider.NewUserProvider(userRepo),
		logger,
	)

	metricsService := metricssvc.NewMetricsQueryService(
		metricsMonitorRepo,
		provider.NewUserProvider(userRepo),
		metricsRepo,
		probeSummaryRepo,
	)

	notificationRegistry := notificationsvc.NewProviderRegistry()
	notificationRegistry.Register(alert.ContactTypeTelegram, notifinfra.NewTelegramNotificationProvider(nil))
	notificationService := notificationsvc.NewNotificationService(
		notificationRegistry,
		eventBus,
		monitoringMonitorRepo,
		logger,
	)

	registry := healthchecksvc.NewProberRegistry()
	registry.Register(target.ProtocolHTTP, infra.NewHTTPProber())

	healthChecker := healthchecksvc.NewHealthChecker(
		healthTargetRepo,
		healthProbeResultRepo,
		eventBus,
		registry,
		healthchecksvc.HealthCheckerConfig{WorkerCount: 5, TaskQueueSize: 100},
		logger,
	)

	evaluator := infra.NewHTTPProbeEvaluator()
	analyzer := analyzationsvc.NewProbeAnalyzationService(
		analyzerMonitorRepo,
		analyzerProbeResultRepo,
		analyzerProbeSummaryRepo,
		evaluator,
		eventBus,
		analyzationsvc.ProbeAnalyzationServiceConfig{-1, -1},
		logger,
	)

	go func() {
		if err := healthChecker.Run(ctx); err != nil {
			logger.Error("health checker stopped", "error", err)
		}
	}()

	go func() {
		if err := analyzer.Run(ctx); err != nil {
			logger.Error("analyzer stopped", "error", err)
		}
	}()

	go func() {
		if err := notificationService.Run(ctx); err != nil {
			logger.Error("notification service stopped", "error", err)
		}
	}()

	runMenu(ctx, authService, monitoringService, maintenanceService, contactService, metricsService)
}

func runMenu(
	ctx context.Context,
	authService authsvc.AuthService,
	monitoringService monitoringsvc.MonitoringManagementService,
	maintenanceService maintenancesvc.MaintenanceService,
	contactService contactssvc.ContactService,
	metricsService metricssvc.MetricQueryService,
) {
	scanner := bufio.NewScanner(os.Stdin)
	currentUser := ""

	for {
		fmt.Println()
		fmt.Println("========== WatchTower CLI: Main ==========")
		if currentUser == "" {
			fmt.Println("User: guest")
		} else {
			fmt.Printf("User: %s\n", currentUser)
		}
		fmt.Println("1) Register")
		fmt.Println("2) Login")
		if currentUser != "" {
			fmt.Println("3) Monitoring")
			fmt.Println("4) Maintenance")
			fmt.Println("5) Contacts")
			fmt.Println("6) Metrics")
			fmt.Println("7) Logout")
		}
		fmt.Println("0) Exit")

		choice, err := prompt(scanner, "Select action: ")
		if err != nil {
			if errorsIsEOF(err) {
				fmt.Println("Input closed. Exiting...")
				return
			}
			fmt.Printf("Input error: %v\n", err)
			continue
		}

		switch choice {
		case "1":
			handleRegister(ctx, scanner, authService)
		case "2":
			currentUser = handleLogin(ctx, scanner, authService, currentUser)
		case "3":
			if currentUser == "" {
				fmt.Println("Please login first")
				continue
			}
			runMonitoringMenu(ctx, scanner, monitoringService, currentUser)
		case "4":
			if currentUser == "" {
				fmt.Println("Please login first")
				continue
			}
			runMaintenanceMenu(ctx, scanner, maintenanceService, currentUser)
		case "5":
			if currentUser == "" {
				fmt.Println("Please login first")
				continue
			}
			runContactsMenu(ctx, scanner, contactService, currentUser)
		case "6":
			if currentUser == "" {
				fmt.Println("Please login first")
				continue
			}
			runMetricsMenu(ctx, scanner, metricsService, currentUser)
		case "7":
			if currentUser == "" {
				fmt.Println("Unknown menu item")
				continue
			}
			currentUser = ""
			fmt.Println("Logged out")
		case "0":
			fmt.Println("Bye")
			return
		default:
			fmt.Println("Unknown menu item")
		}
	}
}

func runMonitoringMenu(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	for {
		fmt.Println()
		fmt.Println("====== WatchTower CLI: Monitoring ======")
		if currentUser == "" {
			fmt.Println("User: guest")
		} else {
			fmt.Printf("User: %s\n", currentUser)
		}
		fmt.Println("1) Create monitor (HTTP)")
		fmt.Println("2) Update monitor (HTTP)")
		fmt.Println("3) Enable monitor")
		fmt.Println("4) Disable monitor")
		fmt.Println("5) Delete monitor")
		fmt.Println("6) Link alert contact")
		fmt.Println("7) Unlink alert contact")
		fmt.Println("8) List my monitors")
		fmt.Println("0) Back")

		choice, err := prompt(scanner, "Select action: ")
		if err != nil {
			if errorsIsEOF(err) {
				fmt.Println("Input closed. Back to main menu...")
				return
			}
			fmt.Printf("Input error: %v\n", err)
			continue
		}

		switch choice {
		case "1":
			handleCreateMonitor(ctx, scanner, monitoringService, currentUser)
		case "2":
			handleUpdateMonitor(ctx, scanner, monitoringService, currentUser)
		case "3":
			handleEnableMonitor(ctx, scanner, monitoringService, currentUser)
		case "4":
			handleDisableMonitor(ctx, scanner, monitoringService, currentUser)
		case "5":
			handleDeleteMonitor(ctx, scanner, monitoringService, currentUser)
		case "6":
			handleLinkAlertContact(ctx, scanner, monitoringService, currentUser)
		case "7":
			handleUnlinkAlertContact(ctx, scanner, monitoringService, currentUser)
		case "8":
			handleListMonitors(ctx, monitoringService, currentUser)
		case "0":
			return
		default:
			fmt.Println("Unknown menu item")
		}
	}
}

func runMaintenanceMenu(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	for {
		fmt.Println()
		fmt.Println("====== WatchTower CLI: Maintenance ======")
		if currentUser == "" {
			fmt.Println("User: guest")
		} else {
			fmt.Printf("User: %s\n", currentUser)
		}
		fmt.Println("1) Create one-time maintenance window")
		fmt.Println("2) Create manual maintenance window")
		fmt.Println("3) Update maintenance window")
		fmt.Println("4) Add monitor to maintenance window")
		fmt.Println("5) Remove monitor from maintenance window")
		fmt.Println("6) Delete maintenance window")
		fmt.Println("7) List my maintenance windows")
		fmt.Println("0) Back")

		choice, err := prompt(scanner, "Select action: ")
		if err != nil {
			if errorsIsEOF(err) {
				fmt.Println("Input closed. Back to main menu...")
				return
			}
			fmt.Printf("Input error: %v\n", err)
			continue
		}

		switch choice {
		case "1":
			handleCreateOneTimeMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "2":
			handleCreateManualMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "3":
			handleUpdateMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "4":
			handleAddMonitorToMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "5":
			handleRemoveMonitorFromMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "6":
			handleDeleteMaintenanceWindow(ctx, scanner, maintenanceService, currentUser)
		case "7":
			handleListMaintenanceWindows(ctx, maintenanceService, currentUser)
		case "0":
			return
		default:
			fmt.Println("Unknown menu item")
		}
	}
}

func runContactsMenu(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	for {
		fmt.Println()
		fmt.Println("====== WatchTower CLI: Contacts ======")
		fmt.Printf("User: %s\n", currentUser)
		fmt.Println("1) Create Telegram contact")
		fmt.Println("2) List contacts")
		fmt.Println("3) Update contact")
		fmt.Println("4) Enable contact")
		fmt.Println("5) Disable contact")
		fmt.Println("6) Delete contact")
		fmt.Println("0) Back")

		choice, err := prompt(scanner, "Select action: ")
		if err != nil {
			if errorsIsEOF(err) {
				fmt.Println("Input closed. Back to main menu...")
				return
			}
			fmt.Printf("Input error: %v\n", err)
			continue
		}

		switch choice {
		case "1":
			handleCreateTelegramContact(ctx, scanner, contactService, currentUser)
		case "2":
			handleListContacts(ctx, contactService, currentUser)
		case "3":
			handleUpdateContact(ctx, scanner, contactService, currentUser)
		case "4":
			handleEnableContact(ctx, scanner, contactService, currentUser)
		case "5":
			handleDisableContact(ctx, scanner, contactService, currentUser)
		case "6":
			handleDeleteContact(ctx, scanner, contactService, currentUser)
		case "0":
			return
		default:
			fmt.Println("Unknown menu item")
		}
	}
}

func runMetricsMenu(
	ctx context.Context,
	scanner *bufio.Scanner,
	metricsService metricssvc.MetricQueryService,
	currentUser string,
) {
	for {
		fmt.Println()
		fmt.Println("====== WatchTower CLI: Metrics ======")
		fmt.Printf("User: %s\n", currentUser)
		fmt.Println("1) Last summaries")
		fmt.Println("2) SLA for period")
		fmt.Println("3) Status history for period")
		fmt.Println("4) Summaries for period")
		fmt.Println("0) Back")

		choice, err := prompt(scanner, "Select action: ")
		if err != nil {
			if errorsIsEOF(err) {
				fmt.Println("Input closed. Back to main menu...")
				return
			}
			fmt.Printf("Input error: %v\n", err)
			continue
		}

		switch choice {
		case "1":
			handleGetLastSummaries(ctx, scanner, metricsService, currentUser)
		case "2":
			handleGetSLA(ctx, scanner, metricsService, currentUser)
		case "3":
			handleGetStatusHistory(ctx, scanner, metricsService, currentUser)
		case "4":
			handleGetSummariesForPeriod(ctx, scanner, metricsService, currentUser)
		case "0":
			return
		default:
			fmt.Println("Unknown menu item")
		}
	}
}

func handleGetSummariesForPeriod(
	ctx context.Context,
	scanner *bufio.Scanner,
	metricsService metricssvc.MetricQueryService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	from, ok := promptTimeRFC3339(scanner, "Period start RFC3339: ")
	if !ok {
		return
	}
	to, ok := promptTimeRFC3339(scanner, "Period end RFC3339: ")
	if !ok {
		return
	}

	summaries, err := metricsService.GetSummaries(authorizedCtx, monitorID, nil, &from, &to)
	if err != nil {
		fmt.Printf("Get summaries for period failed: %v\n", err)
		return
	}

	if len(summaries) == 0 {
		fmt.Println("No summaries found")
		return
	}

	for _, s := range summaries {
		fmt.Printf("- time=%s status=%s latency=%dms network_failure=%t status_code=%d\n",
			s.ProbeTime.Format(time.RFC3339),
			s.MonitorStatus,
			s.LatencyMs,
			s.NetworkFailure,
			s.StatusCode,
		)
	}
}

func handleRegister(ctx context.Context, scanner *bufio.Scanner, authService authsvc.AuthService) {
	login, err := prompt(scanner, "Login: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	password, err := prompt(scanner, "Password: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	if err := authService.Register(ctx, login, password); err != nil {
		fmt.Printf("Register failed: %v\n", err)
		return
	}

	fmt.Println("User registered")
}

func handleLogin(
	ctx context.Context,
	scanner *bufio.Scanner,
	authService authsvc.AuthService,
	currentUser string,
) string {
	login, err := prompt(scanner, "Login: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return currentUser
	}
	password, err := prompt(scanner, "Password: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return currentUser
	}

	token, err := authService.Login(ctx, login, password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return currentUser
	}

	authorizedUser, err := authService.ParseToken(ctx, token)
	if err != nil {
		fmt.Printf("Token parse failed: %v\n", err)
		return currentUser
	}

	fmt.Printf("Login successful. JWT: %s\n", token)
	return authorizedUser
}

func handleCreateMonitor(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	label, err := prompt(scanner, "Label: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	endpoint, err := prompt(scanner, "Endpoint (e.g. https://google.com): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	probeIntervalSec, ok := promptInt32(scanner, "Probe interval sec: ")
	if !ok {
		return
	}
	method, err := prompt(scanner, "HTTP method [GET]: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	if method == "" {
		method = "GET"
	}
	maxLatencyMs, ok := promptInt(scanner, "Max latency ms: ")
	if !ok {
		return
	}
	statusCodes, ok := promptCSVInts(scanner, "Expected status codes (comma separated, e.g. 200,301): ")
	if !ok {
		return
	}

	err = monitoringService.CreateMonitor(authorizedCtx, dto.CreateMonitorDTO{
		Label:            label,
		Endpoint:         endpoint,
		ProbeIntervalSec: probeIntervalSec,
		NetworkConfig: dto.HTTPMonitorNetworkConfig{
			Method: method,
		},
		Expectations: dto.HTTPMonitorExpectations{
			MaxLatencyMs: maxLatencyMs,
			StatusCodes:  statusCodes,
		},
	})
	if err != nil {
		fmt.Printf("Create monitor failed: %v\n", err)
		return
	}

	fmt.Println("Monitor created")
}

func handleUpdateMonitor(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}

	labelRaw, err := prompt(scanner, "New label (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	endpointRaw, err := prompt(scanner, "New endpoint (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	probeRaw, err := prompt(scanner, "New probe interval sec (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	methodRaw, err := prompt(scanner, "New HTTP method (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	latencyRaw, err := prompt(scanner, "New max latency ms (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	statusCodesRaw, err := prompt(scanner, "New expected status codes (comma separated, empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	updateDTO := dto.UpdateMonitorDTO{ID: monitorID}

	if labelRaw != "" {
		label := labelRaw
		updateDTO.Label = &label
	}

	if endpointRaw != "" {
		endpoint := endpointRaw
		updateDTO.Endpoint = &endpoint
	}

	if probeRaw != "" {
		probeInt, err := strconv.ParseInt(probeRaw, 10, 32)
		if err != nil {
			fmt.Printf("Invalid probe interval: %v\n", err)
			return
		}
		probeInterval := int32(probeInt)
		updateDTO.ProbeIntervalSec = &probeInterval
	}

	needProtocol := false
	if methodRaw != "" {
		networkCfg := dto.MonitorNetworkConfig(dto.HTTPMonitorNetworkConfig{Method: methodRaw})
		updateDTO.NetworkConfig = &networkCfg
		needProtocol = true
	}

	if latencyRaw != "" || statusCodesRaw != "" {
		latency := 0
		if latencyRaw != "" {
			latency, err = strconv.Atoi(latencyRaw)
			if err != nil {
				fmt.Printf("Invalid max latency: %v\n", err)
				return
			}
		}

		statusCodes := []int{}
		if statusCodesRaw != "" {
			statusCodes, ok = parseCSVInts(statusCodesRaw)
			if !ok {
				fmt.Println("Invalid status codes list")
				return
			}
		}

		expectations := dto.MonitorExpectations(dto.HTTPMonitorExpectations{
			MaxLatencyMs: latency,
			StatusCodes:  statusCodes,
		})
		updateDTO.Expectations = &expectations
		needProtocol = true
	}

	if needProtocol {
		protocol := target.ProtocolHTTP
		updateDTO.Protocol = &protocol
	}

	if err := monitoringService.UpdateMonitor(authorizedCtx, updateDTO); err != nil {
		fmt.Printf("Update monitor failed: %v\n", err)
		return
	}

	fmt.Println("Monitor updated")
}

func handleEnableMonitor(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}

	if err := monitoringService.EnableMonitor(authorizedCtx, monitorID); err != nil {
		fmt.Printf("Enable monitor failed: %v\n", err)
		return
	}

	fmt.Println("Monitor enabled")
}

func handleDisableMonitor(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}

	if err := monitoringService.DisableMonitor(authorizedCtx, monitorID); err != nil {
		fmt.Printf("Disable monitor failed: %v\n", err)
		return
	}

	fmt.Println("Monitor disabled")
}

func handleDeleteMonitor(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}

	if err := monitoringService.DeleteMonitor(authorizedCtx, monitorID); err != nil {
		fmt.Printf("Delete monitor failed: %v\n", err)
		return
	}

	fmt.Println("Monitor deleted")
}

func handleListMonitors(
	ctx context.Context,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitors, err := monitoringService.GetAllMonitors(authorizedCtx)
	if err != nil {
		fmt.Printf("Get monitors failed: %v\n", err)
		return
	}

	if len(monitors) == 0 {
		fmt.Println("No monitors found")
		return
	}

	for _, mon := range monitors {
		fmt.Printf("- id=%s label=%s status=%s active=%t endpoint=%s interval=%ds\n",
			mon.ID,
			mon.Label,
			mon.CurrentStatus,
			mon.IsActive,
			mon.Target.Endpoint,
			mon.ProbeIntervalSec,
		)
	}
}

func handleLinkAlertContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	contactID, ok := promptUUID(scanner, "Alert contact ID: ")
	if !ok {
		return
	}

	if err := monitoringService.LinkAlertContact(authorizedCtx, monitorID, contactID); err != nil {
		fmt.Printf("Link alert contact failed: %v\n", err)
		return
	}

	fmt.Println("Alert contact linked")
}

func handleUnlinkAlertContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	monitoringService monitoringsvc.MonitoringManagementService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	contactID, ok := promptUUID(scanner, "Alert contact ID: ")
	if !ok {
		return
	}

	if err := monitoringService.UnlinkAlertContact(authorizedCtx, monitorID, contactID); err != nil {
		fmt.Printf("Unlink alert contact failed: %v\n", err)
		return
	}

	fmt.Println("Alert contact unlinked")
}

func handleCreateOneTimeMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	title, err := prompt(scanner, "Title: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	description, err := prompt(scanner, "Description: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	startTime, ok := promptTimeRFC3339(scanner, "Start time RFC3339 (e.g. 2026-04-17T15:04:05Z): ")
	if !ok {
		return
	}
	endTime, ok := promptTimeRFC3339(scanner, "End time RFC3339 (e.g. 2026-04-17T16:04:05Z): ")
	if !ok {
		return
	}

	window, err := maintenanceService.CreateOneTimeMaintenanceWindow(authorizedCtx, maintenancesvc.CreateOneTimeMaintenanceWindowDTO{
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
	})
	if err != nil {
		fmt.Printf("Create one-time maintenance window failed: %v\n", err)
		return
	}

	fmt.Printf("One-time maintenance window created: %s\n", window.ID)
}

func handleCreateManualMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	title, err := prompt(scanner, "Title: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	description, err := prompt(scanner, "Description: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	window, err := maintenanceService.CreateManualMaintenanceWindow(authorizedCtx, maintenancesvc.CreateManualMaintenanceWindowDTO{
		Title:       title,
		Description: description,
	})
	if err != nil {
		fmt.Printf("Create manual maintenance window failed: %v\n", err)
		return
	}

	fmt.Printf("Manual maintenance window created: %s\n", window.ID)
}

func handleUpdateMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	windowID, ok := promptUUID(scanner, "Maintenance window ID: ")
	if !ok {
		return
	}

	titleRaw, err := prompt(scanner, "New title (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	descriptionRaw, err := prompt(scanner, "New description (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	configTypeRaw, err := prompt(scanner, "Config update type [none|one-time|manual]: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	updateDTO := maintenancesvc.UpdateMaintenanceWindowDTO{WindowID: windowID}
	if titleRaw != "" {
		title := titleRaw
		updateDTO.Title = &title
	}
	if descriptionRaw != "" {
		description := descriptionRaw
		updateDTO.Description = &description
	}

	switch strings.ToLower(strings.TrimSpace(configTypeRaw)) {
	case "", "none":
		// Keep config unchanged.
	case "one-time":
		startRaw, err := prompt(scanner, "New start time RFC3339 (empty to keep): ")
		if err != nil {
			fmt.Printf("Input error: %v\n", err)
			return
		}
		endRaw, err := prompt(scanner, "New end time RFC3339 (empty to keep): ")
		if err != nil {
			fmt.Printf("Input error: %v\n", err)
			return
		}

		var startPtr *time.Time
		if startRaw != "" {
			startTime, err := time.Parse(time.RFC3339, startRaw)
			if err != nil {
				fmt.Printf("Invalid start time: %v\n", err)
				return
			}
			startPtr = &startTime
		}

		var endPtr *time.Time
		if endRaw != "" {
			endTime, err := time.Parse(time.RFC3339, endRaw)
			if err != nil {
				fmt.Printf("Invalid end time: %v\n", err)
				return
			}
			endPtr = &endTime
		}

		updateDTO.ConfigUpdate = maintenance.OneTimeConfigUpdate{
			StartTime: startPtr,
			EndTime:   endPtr,
		}
	case "manual":
		activeRaw, err := prompt(scanner, "Set active [true|false]: ")
		if err != nil {
			fmt.Printf("Input error: %v\n", err)
			return
		}
		active, err := strconv.ParseBool(strings.TrimSpace(activeRaw))
		if err != nil {
			fmt.Printf("Invalid bool value: %v\n", err)
			return
		}

		updateDTO.ConfigUpdate = maintenance.ManualConfigUpdate{Active: &active}
	default:
		fmt.Println("Unknown config update type")
		return
	}

	if err := maintenanceService.UpdateMaintenanceWindow(authorizedCtx, updateDTO); err != nil {
		fmt.Printf("Update maintenance window failed: %v\n", err)
		return
	}

	fmt.Println("Maintenance window updated")
}

func handleAddMonitorToMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	windowID, ok := promptUUID(scanner, "Maintenance window ID: ")
	if !ok {
		return
	}

	if err := maintenanceService.AddMonitorToMaintenanceWindow(authorizedCtx, monitorID, windowID); err != nil {
		fmt.Printf("Add monitor to maintenance window failed: %v\n", err)
		return
	}

	fmt.Println("Monitor added to maintenance window")
}

func handleRemoveMonitorFromMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	windowID, ok := promptUUID(scanner, "Maintenance window ID: ")
	if !ok {
		return
	}

	if err := maintenanceService.RemoveMonitorFromMaintenanceWindow(authorizedCtx, monitorID, windowID); err != nil {
		fmt.Printf("Remove monitor from maintenance window failed: %v\n", err)
		return
	}

	fmt.Println("Monitor removed from maintenance window")
}

func handleDeleteMaintenanceWindow(
	ctx context.Context,
	scanner *bufio.Scanner,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	windowID, ok := promptUUID(scanner, "Maintenance window ID: ")
	if !ok {
		return
	}

	if err := maintenanceService.DeleteMaintenanceWindow(authorizedCtx, windowID); err != nil {
		fmt.Printf("Delete maintenance window failed: %v\n", err)
		return
	}

	fmt.Println("Maintenance window deleted")
}

func handleListMaintenanceWindows(
	ctx context.Context,
	maintenanceService maintenancesvc.MaintenanceService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	windows, err := maintenanceService.GetAllMaintenanceWindows(authorizedCtx)
	if err != nil {
		fmt.Printf("Get maintenance windows failed: %v\n", err)
		return
	}

	if len(windows) == 0 {
		fmt.Println("No maintenance windows found")
		return
	}

	for _, window := range windows {
		fmt.Printf("- id=%s title=%s type=%s active=%t\n",
			window.ID,
			window.Title,
			window.Type,
			window.IsActive(),
		)
	}
}

func handleGetLastSummaries(
	ctx context.Context,
	scanner *bufio.Scanner,
	metricsService metricssvc.MetricQueryService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	limit, ok := promptInt(scanner, "Limit: ")
	if !ok {
		return
	}

	summaries, err := metricsService.GetSummaries(authorizedCtx, monitorID, &limit, nil, nil)
	if err != nil {
		fmt.Printf("Get summaries failed: %v\n", err)
		return
	}

	if len(summaries) == 0 {
		fmt.Println("No summaries found")
		return
	}

	for _, s := range summaries {
		fmt.Printf("- time=%s status=%s latency=%dms network_failure=%t status_code=%d\n",
			s.ProbeTime.Format(time.RFC3339),
			s.MonitorStatus,
			s.LatencyMs,
			s.NetworkFailure,
			s.StatusCode,
		)
	}
}

func handleGetSLA(
	ctx context.Context,
	scanner *bufio.Scanner,
	metricsService metricssvc.MetricQueryService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	from, ok := promptTimeRFC3339(scanner, "Period start RFC3339: ")
	if !ok {
		return
	}
	to, ok := promptTimeRFC3339(scanner, "Period end RFC3339: ")
	if !ok {
		return
	}

	sla, err := metricsService.GetSLA(authorizedCtx, monitorID, from, to)
	if err != nil {
		fmt.Printf("Get SLA failed: %v\n", err)
		return
	}

	fmt.Printf("SLA: monitor_id=%s uptime=%.2f%% downtime=%ds period=[%s..%s]\n",
		sla.MonitorID,
		sla.UptimePercent,
		sla.TotalDowntime,
		sla.PeriodStart.Format(time.RFC3339),
		sla.PeriodEnd.Format(time.RFC3339),
	)
}

func handleGetStatusHistory(
	ctx context.Context,
	scanner *bufio.Scanner,
	metricsService metricssvc.MetricQueryService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	monitorID, ok := promptUUID(scanner, "Monitor ID: ")
	if !ok {
		return
	}
	from, ok := promptTimeRFC3339(scanner, "Period start RFC3339: ")
	if !ok {
		return
	}
	to, ok := promptTimeRFC3339(scanner, "Period end RFC3339: ")
	if !ok {
		return
	}

	history, err := metricsService.GetStatusHistory(authorizedCtx, monitorID, from, to)
	if err != nil {
		fmt.Printf("Get status history failed: %v\n", err)
		return
	}

	if len(history) == 0 {
		fmt.Println("No status events found")
		return
	}

	for _, e := range history {
		fmt.Printf("- status=%s start=%s end=%s\n",
			e.Status,
			e.StartTime.Format(time.RFC3339),
			e.EndTime.Format(time.RFC3339),
		)
	}
}

func handleCreateTelegramContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	name, err := prompt(scanner, "Contact name: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	chatID, ok := promptInt64(scanner, "Telegram chat id: ")
	if !ok {
		return
	}
	botToken, err := prompt(scanner, "Telegram bot token: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	contact, err := contactService.CreateTelegramAlertContact(authorizedCtx, contact_dto.CreateTelegramAlertContactDTO{
		Name:     name,
		ChatID:   chatID,
		BotToken: botToken,
	})
	if err != nil {
		fmt.Printf("Create contact failed: %v\n", err)
		return
	}

	fmt.Printf("Contact created: %s\n", contact.ID)
}

func handleListContacts(
	ctx context.Context,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	contacts, err := contactService.GetAllAlertContacts(authorizedCtx)
	if err != nil {
		fmt.Printf("Get contacts failed: %v\n", err)
		return
	}

	if len(contacts) == 0 {
		fmt.Println("No contacts found")
		return
	}

	for _, c := range contacts {
		fmt.Printf("- id=%s name=%s type=%s active=%t", c.ID, c.Name, c.Type, c.IsActive)
		if cfg, ok := c.Config.(alert.TelegramContactConfig); ok {
			fmt.Printf(" chat_id=%d", cfg.ChatID)
		}
		fmt.Println()
	}
}

func handleUpdateContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	contactID, ok := promptUUID(scanner, "Contact ID: ")
	if !ok {
		return
	}

	nameRaw, err := prompt(scanner, "New name (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	isActiveRaw, err := prompt(scanner, "Set active [true|false|empty to keep]: ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	chatIDRaw, err := prompt(scanner, "New telegram chat id (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}
	botTokenRaw, err := prompt(scanner, "New telegram bot token (empty to keep): ")
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return
	}

	upd := contact_dto.UpdateAlertContactDTO{ContactID: contactID}
	if nameRaw != "" {
		name := nameRaw
		upd.Name = &name
	}
	if isActiveRaw != "" {
		isActive, err := strconv.ParseBool(strings.TrimSpace(isActiveRaw))
		if err != nil {
			fmt.Printf("Invalid bool value: %v\n", err)
			return
		}
		upd.IsActive = &isActive
	}

	var cfgUpd alert.TelegramConfigUpdate
	hasCfgUpdate := false
	if chatIDRaw != "" {
		chatID, err := strconv.ParseInt(strings.TrimSpace(chatIDRaw), 10, 64)
		if err != nil {
			fmt.Printf("Invalid chat id: %v\n", err)
			return
		}
		cfgUpd.ChatID = &chatID
		hasCfgUpdate = true
	}
	if botTokenRaw != "" {
		botToken := botTokenRaw
		cfgUpd.BotToken = &botToken
		hasCfgUpdate = true
	}
	if hasCfgUpdate {
		upd.ConfigUpdate = cfgUpd
	}

	if err := contactService.UpdateAlertContact(authorizedCtx, upd); err != nil {
		fmt.Printf("Update contact failed: %v\n", err)
		return
	}

	fmt.Println("Contact updated")
}

func handleEnableContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	contactID, ok := promptUUID(scanner, "Contact ID: ")
	if !ok {
		return
	}

	if err := contactService.EnableAlertContact(authorizedCtx, contactID); err != nil {
		fmt.Printf("Enable contact failed: %v\n", err)
		return
	}

	fmt.Println("Contact enabled")
}

func handleDisableContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	contactID, ok := promptUUID(scanner, "Contact ID: ")
	if !ok {
		return
	}

	if err := contactService.DisableAlertContact(authorizedCtx, contactID); err != nil {
		fmt.Printf("Disable contact failed: %v\n", err)
		return
	}

	fmt.Println("Contact disabled")
}

func handleDeleteContact(
	ctx context.Context,
	scanner *bufio.Scanner,
	contactService contactssvc.ContactService,
	currentUser string,
) {
	authorizedCtx, ok := userCtx(ctx, currentUser)
	if !ok {
		fmt.Println("Please login first")
		return
	}

	contactID, ok := promptUUID(scanner, "Contact ID: ")
	if !ok {
		return
	}

	if err := contactService.DeleteAlertContact(authorizedCtx, contactID); err != nil {
		fmt.Printf("Delete contact failed: %v\n", err)
		return
	}

	fmt.Println("Contact deleted")
}

func userCtx(base context.Context, currentUser string) (context.Context, bool) {
	if currentUser == "" {
		return nil, false
	}

	return authsvc.ContextWithUser(base, currentUser), true
}

func prompt(scanner *bufio.Scanner, label string) (string, error) {
	fmt.Print(label)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}

	return strings.TrimSpace(scanner.Text()), nil
}

func promptInt(scanner *bufio.Scanner, label string) (int, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return 0, false
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		fmt.Printf("Invalid integer: %v\n", err)
		return 0, false
	}

	return intVal, true
}

func promptInt32(scanner *bufio.Scanner, label string) (int32, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return 0, false
	}

	intVal, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		fmt.Printf("Invalid int32: %v\n", err)
		return 0, false
	}

	return int32(intVal), true
}

func promptInt64(scanner *bufio.Scanner, label string) (int64, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return 0, false
	}

	intVal, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		fmt.Printf("Invalid int64: %v\n", err)
		return 0, false
	}

	return intVal, true
}

func promptUUID(scanner *bufio.Scanner, label string) (uuid.UUID, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return uuid.Nil, false
	}

	id, err := uuid.Parse(value)
	if err != nil {
		fmt.Printf("Invalid UUID: %v\n", err)
		return uuid.Nil, false
	}

	return id, true
}

func promptCSVInts(scanner *bufio.Scanner, label string) ([]int, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return nil, false
	}

	values, ok := parseCSVInts(value)
	if !ok {
		fmt.Println("Invalid integer list")
		return nil, false
	}

	return values, true
}

func parseCSVInts(raw string) ([]int, bool) {
	parts := strings.Split(raw, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		result = append(result, value)
	}

	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}

func promptTimeRFC3339(scanner *bufio.Scanner, label string) (time.Time, bool) {
	value, err := prompt(scanner, label)
	if err != nil {
		fmt.Printf("Input error: %v\n", err)
		return time.Time{}, false
	}

	timeVal, err := time.Parse(time.RFC3339, value)
	if err != nil {
		fmt.Printf("Invalid time format. Expected RFC3339: %v\n", err)
		return time.Time{}, false
	}

	return timeVal, true
}

type filteredWatermillLogger struct {
	next watermill.LoggerAdapter
}

func newFilteredWatermillLogger(next watermill.LoggerAdapter) watermill.LoggerAdapter {
	return &filteredWatermillLogger{next: next}
}

func (l *filteredWatermillLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.next.Error(msg, err, fields)
}

func (l *filteredWatermillLogger) Info(msg string, fields watermill.LogFields) {
	if msg == "No subscribers to send message" {
		if topic, ok := fields["topic"].(string); ok && topic == analyzationsvc.TopicMonitorStatusChanged {
			return
		}
	}

	l.next.Info(msg, fields)
}

func (l *filteredWatermillLogger) Debug(msg string, fields watermill.LogFields) {
	l.next.Debug(msg, fields)
}

func (l *filteredWatermillLogger) Trace(msg string, fields watermill.LogFields) {
	l.next.Trace(msg, fields)
}

func (l *filteredWatermillLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &filteredWatermillLogger{next: l.next.With(fields)}
}
