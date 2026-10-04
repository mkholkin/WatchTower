// Lab 3 measures JSON operations on the application's HTTP response type.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"

	api "WatchTower/internal/api/http/v1/gen"
	fastjson "github.com/goccy/go-json"
	"github.com/google/uuid"
)

type library struct {
	name      string
	marshal   func(any) ([]byte, error)
	unmarshal func([]byte, any) error
}

var libraries = []library{{"encoding-json", json.Marshal, json.Unmarshal}, {"goccy-go-json", fastjson.Marshal, fastjson.Unmarshal}}
var sinkBytes []byte
var sinkChecks []api.MonitorCheck
var sinkCounter uint64

type config struct {
	Warmup         int `json:"warmup"`
	Batches        int `json:"batches"`
	BatchSize      int `json:"batch_size"`
	LatencySamples int `json:"latency_samples"`
	Teardown       int `json:"teardown"`
}
type phase struct {
	Operations int   `json:"operations"`
	ElapsedNS  int64 `json:"elapsed_ns"`
}
type batch struct {
	OffsetNS int64   `json:"offset_ns"`
	NSPerOp  float64 `json:"ns_per_op"`
}
type caseResult struct {
	ID              string    `json:"id"`
	Library         string    `json:"library"`
	Scenario        string    `json:"scenario"`
	Size            int       `json:"size"`
	PayloadBytes    int       `json:"payload_bytes"`
	StartedAt       time.Time `json:"started_at"`
	Warmup          phase     `json:"warmup"`
	Measurement     phase     `json:"measurement"`
	Latency         phase     `json:"latency"`
	Teardown        phase     `json:"teardown"`
	Batches         []batch   `json:"batches"`
	LatenciesNS     []int64   `json:"latencies_ns"`
	TotalOperations int       `json:"total_operations"`
	DurationNS      int64     `json:"duration_ns"`
}
type trialResult struct {
	Trial            int               `json:"trial"`
	Config           config            `json:"config"`
	GoVersion        string            `json:"go_version"`
	GOOS             string            `json:"goos"`
	GOARCH           string            `json:"goarch"`
	GOMAXPROCS       int               `json:"gomaxprocs"`
	Dependencies     map[string]string `json:"dependencies"`
	StartedAt        time.Time         `json:"started_at"`
	DurationNS       int64             `json:"duration_ns"`
	OverheadNSPerOp  float64           `json:"overhead_ns_per_op"`
	TimerProbeNS     []int64           `json:"timer_probe_ns"`
	TimerPairNSPerOp float64           `json:"timer_pair_ns_per_op"`
	Cases            []caseResult      `json:"cases"`
}

func fixture(size int) []api.MonitorCheck {
	data := make([]api.MonitorCheck, size)
	for i := range data {
		code := 200
		data[i] = api.MonitorCheck{CheckTime: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC), MonitorId: uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef"), Status: api.Up, LatencyMs: 10 + i, StatusCode: &code}
		if i%3 == 2 {
			reason := "HTTP 503: сервер недоступен <retry>"
			code = 503
			data[i].Status = api.Down
			data[i].FailureReason = &reason
		}
	}
	return data
}

func prepare(lib library, scenario string, data []api.MonitorCheck) (func() error, error) {
	// Equivalence checks and input serialization happen outside every timing window.
	canonical, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	encoded, err := lib.marshal(data)
	if err != nil {
		return nil, err
	}
	var decoded []api.MonitorCheck
	if err = lib.unmarshal(encoded, &decoded); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(data, decoded) || string(encoded) != string(canonical) {
		return nil, fmt.Errorf("%s: JSON implementations disagree", lib.name)
	}
	switch scenario {
	case "encode":
		return func() error { var err error; sinkBytes, err = lib.marshal(data); return err }, nil
	case "decode":
		return func() error {
			var result []api.MonitorCheck
			err := lib.unmarshal(canonical, &result)
			sinkChecks = result
			return err
		}, nil
	case "roundtrip":
		return func() error {
			var err error
			sinkBytes, err = lib.marshal(data)
			if err != nil {
				return err
			}
			var result []api.MonitorCheck
			err = lib.unmarshal(sinkBytes, &result)
			sinkChecks = result
			return err
		}, nil
	default:
		return nil, fmt.Errorf("unknown scenario %q", scenario)
	}
}

//go:noinline
func control() error { sinkCounter++; return nil }

//go:noinline
func clockControl() error {
	start := time.Now()
	sinkCounter += uint64(time.Since(start).Nanoseconds())
	return nil
}

func measure(n int, op func() error) (phase, error) {
	start := time.Now()
	for i := 0; i < n; i++ {
		if err := op(); err != nil {
			return phase{}, err
		}
	}
	return phase{Operations: n, ElapsedNS: time.Since(start).Nanoseconds()}, nil
}

func runCase(lib library, scenario string, size int, cfg config) (caseResult, error) {
	start := time.Now()
	data := fixture(size)
	payload, _ := json.Marshal(data)
	r := caseResult{ID: fmt.Sprintf("%s/%s/%d", lib.name, scenario, size), Library: lib.name, Scenario: scenario, Size: size, PayloadBytes: len(payload), StartedAt: start.UTC()}
	op, err := prepare(lib, scenario, data)
	if err != nil {
		return r, err
	}
	runtime.GC() // Same pre-warmup policy; automatic GC remains enabled during measurement.
	if r.Warmup, err = measure(cfg.Warmup, op); err != nil {
		return r, err
	}
	for i := 0; i < cfg.Batches; i++ {
		offset := time.Since(start).Nanoseconds()
		p, e := measure(cfg.BatchSize, op)
		if e != nil {
			return r, e
		}
		r.Batches = append(r.Batches, batch{offset, float64(p.ElapsedNS) / float64(p.Operations)})
		r.Measurement.Operations += p.Operations
		r.Measurement.ElapsedNS += p.ElapsedNS
	}
	// Individual-request quantiles are separate: these include two clock reads.
	for i := 0; i < cfg.LatencySamples; i++ {
		p, e := measure(1, op)
		if e != nil {
			return r, e
		}
		r.LatenciesNS = append(r.LatenciesNS, p.ElapsedNS)
		r.Latency.Operations++
		r.Latency.ElapsedNS += p.ElapsedNS
	}
	if r.Teardown, err = measure(cfg.Teardown, op); err != nil {
		return r, err
	}
	r.TotalOperations = r.Warmup.Operations + r.Measurement.Operations + r.Latency.Operations + r.Teardown.Operations
	sinkBytes = nil
	sinkChecks = nil
	r.DurationNS = time.Since(start).Nanoseconds()
	return r, nil
}

func runTrial(trial int, cfg config) (trialResult, error) {
	start := time.Now()
	r := trialResult{Trial: trial, Config: cfg, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GOMAXPROCS: runtime.GOMAXPROCS(0), StartedAt: start.UTC(), Dependencies: map[string]string{}}
	if trial < 1 || cfg.Warmup < 1 || cfg.Batches < 1 || cfg.BatchSize < 1 || cfg.LatencySamples < 1 || cfg.Teardown < 1 {
		return r, fmt.Errorf("trial and phase counts must be positive")
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, d := range info.Deps {
			r.Dependencies[d.Path] = d.Version
		}
	}
	// 1 million calls amortize timer cost; the control keeps loop/dispatch/error checks.
	if _, err := measure(10000, control); err != nil {
		return r, err
	}
	p, err := measure(1000000, control)
	if err != nil {
		return r, err
	}
	r.OverheadNSPerOp = float64(p.ElapsedNS) / float64(p.Operations)
	clock, err := measure(100000, clockControl)
	if err != nil {
		return r, err
	}
	r.TimerPairNSPerOp = float64(clock.ElapsedNS) / float64(clock.Operations)
	for i := 0; i < 256; i++ {
		p, _ := measure(1, control)
		r.TimerProbeNS = append(r.TimerProbeNS, p.ElapsedNS)
	}
	type task struct {
		lib      library
		scenario string
		size     int
	}
	var tasks []task
	for _, lib := range libraries {
		for _, s := range []string{"encode", "decode", "roundtrip"} {
			for _, n := range []int{1, 32} {
				tasks = append(tasks, task{lib, s, n})
			}
		}
	}
	// Reproducible counterbalancing avoids always timing one implementation first.
	rand.New(rand.NewSource(int64(trial))).Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })
	for _, task := range tasks {
		c, err := runCase(task.lib, task.scenario, task.size, cfg)
		if err != nil {
			return r, err
		}
		r.Cases = append(r.Cases, c)
	}
	r.DurationNS = time.Since(start).Nanoseconds()
	return r, nil
}

func main() {
	trial := flag.Int("trial", 1, "independent trial number and order seed")
	cfg := config{}
	flag.IntVar(&cfg.Warmup, "warmup", 256, "warmup operations per case")
	flag.IntVar(&cfg.Batches, "batches", 16, "measured batches per case")
	flag.IntVar(&cfg.BatchSize, "batch-size", 64, "operations per measured batch")
	flag.IntVar(&cfg.LatencySamples, "latency-samples", 256, "individually timed requests per case")
	flag.IntVar(&cfg.Teardown, "teardown", 64, "cooldown operations excluded from measurement")
	flag.Parse()
	runtime.GOMAXPROCS(1)
	r, err := runTrial(*trial, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
