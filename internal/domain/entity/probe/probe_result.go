package probe

import (
	"WatchTower/internal/domain/entity/target"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ProcessingStatus string

const (
	ProcessingStatusProcessed ProcessingStatus = "processed"
	ProcessingStatusNew       ProcessingStatus = "new"
	ProcessingStatusCanceled  ProcessingStatus = "canceled"
)

type Result struct {
	ID               uuid.UUID
	LatencyMs        int32
	Meta             []byte
	NetworkFailure   bool
	ErrorMessage     *string // TODO: унифицировать nullable поля
	StatusCode       sql.NullInt32
	Target           *target.Target
	ProbeTime        time.Time
	ProcessingStatus ProcessingStatus
}

func NewProbeResult(tgt target.Target, latencyMs int32, statusCode int32, meta []byte) (*Result, error) {
	if latencyMs < 0 {
		return nil, errors.New("latency ms must be positive")
	}

	if statusCode < 0 {
		return nil, errors.New("status code must be positive")
	}

	return &Result{
		ID:               uuid.New(),
		Target:           &tgt,
		NetworkFailure:   false,
		LatencyMs:        latencyMs,
		StatusCode:       sql.NullInt32{Int32: statusCode, Valid: true},
		Meta:             meta,
		ProbeTime:        time.Now(),
		ProcessingStatus: ProcessingStatusNew,
	}, nil
}

func NewProbeResultWithNetworkFailure(tgt *target.Target, latencyMs int32, errorMsg string) (*Result, error) {
	if latencyMs < 0 {
		return nil, errors.New("latency ms must be positive")
	}

	return &Result{
		ID:               uuid.New(),
		NetworkFailure:   true,
		LatencyMs:        latencyMs,
		Target:           tgt,
		ProbeTime:        time.Now(),
		ProcessingStatus: ProcessingStatusNew,
		ErrorMessage:     &errorMsg,
	}, nil
}
