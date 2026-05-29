package infra

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"context"
)

type TCPProber struct {
}

func (T TCPProber) Probe(ctx context.Context, target *target.Target) (probe.Result, error) {
	//TODO implement me
	panic("implement me")
}
