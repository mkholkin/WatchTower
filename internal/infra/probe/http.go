package infra

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	analyzationsvc "WatchTower/internal/service/analyze"
	healthchecksvc "WatchTower/internal/service/healthcheck"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

type httpProber struct {
	client *http.Client
}

func NewHTTPProber() healthchecksvc.Prober {
	return &httpProber{
		client: &http.Client{
			Timeout: time.Second * 30, //TODO: вынести в конфиг
		},
	}
}

func (p *httpProber) Probe(ctx context.Context, tgt *target.Target) (*probe.Result, error) {
	req, err := requestFromTarget(ctx, tgt)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(startTime)
	if err != nil {
		return probe.NewProbeResultWithNetworkFailure(
			tgt, int32(latency.Milliseconds()), err.Error())
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body) // todo: положить тело в базу
	if err != nil {
		return nil, err
	}

	res, err := probe.NewProbeResult(
		*tgt, int32(latency.Milliseconds()), int32(resp.StatusCode), nil)

	//res.Meta = ""
	return res, err
}

func toHttpMethod(method string) string {
	switch method {
	case "GET":
		return http.MethodGet
	case "POST":
		return http.MethodPost
	case "PUT":
		return http.MethodPut
	case "DELETE":
		return http.MethodDelete
	case "HEAD":
		return http.MethodHead
	case "OPTIONS":
		return http.MethodOptions
	case "TRACE":
		return http.MethodTrace
	default:
		return method
	}
}

func requestFromTarget(ctx context.Context, tgt *target.Target) (*http.Request, error) {
	config, ok := tgt.Config.(target.HTTPConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}

	req, err := http.NewRequestWithContext(
		ctx, toHttpMethod(config.Method), tgt.Endpoint, strings.NewReader(config.Body),
	)
	if err != nil {
		return nil, err
	}

	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// Evaluator

type httpProbeEvaluator struct{}

func NewHTTPProbeEvaluator() analyzationsvc.ProbeEvaluator {
	return &httpProbeEvaluator{}
}

func (e *httpProbeEvaluator) Evaluate(
	ctx context.Context,
	probeResult *probe.Result,
	mon *monitor.Monitor,
) (monitor.Status, error) {
	expectations, ok := mon.Expectations.(monitor.HTTPExpectations)
	if !ok {
		return monitor.StatusUnknown, errors.New("invalid expectations type")
	}

	if probeResult.NetworkFailure {
		return monitor.StatusDown, nil
	}

	if int(probeResult.LatencyMs) > expectations.MaxLatencyMs {
		return monitor.StatusDown, nil
	}

	if !slices.Contains(expectations.StatusCodes, int(probeResult.StatusCode.Int32)) {
		return monitor.StatusDown, nil
	}

	return monitor.StatusUp, nil
}
