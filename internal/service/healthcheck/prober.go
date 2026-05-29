package healtcheck_service

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"context"
	"fmt"
)

// Prober performs a network probe against a target.
type Prober interface {
	Probe(ctx context.Context, target *target.Target) (*probe.Result, error)
}

type ProberRegistry interface {
	Register(protocol target.Protocol, prober Prober)
	Get(protocol target.Protocol) (Prober, error)
}

// proberRegistryImpl maps protocols to their Prober implementations.
// To add a new protocol, register a new Prober.
type proberRegistryImpl struct {
	probers map[target.Protocol]Prober
}

// NewProberRegistry creates an empty ProberRegistry.
func NewProberRegistry() ProberRegistry {
	return &proberRegistryImpl{probers: make(map[target.Protocol]Prober)}
}

// Register adds a Prober for the given protocol.
func (r *proberRegistryImpl) Register(protocol target.Protocol, prober Prober) {
	r.probers[protocol] = prober
}

// Get returns the Prober for the given protocol or an error if none is registered.
func (r *proberRegistryImpl) Get(protocol target.Protocol) (Prober, error) {
	p, ok := r.probers[protocol]
	if !ok {
		return nil, fmt.Errorf("no prober registered for protocol %s", protocol)
	}
	return p, nil
}
