package service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestProberRegistry_RegisterAndGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registry := NewProberRegistry()
	expected := testmocks.NewMockProber(ctrl)
	registry.Register(target.ProtocolHTTP, expected)

	actual, err := registry.Get(target.ProtocolHTTP)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected prober instance")
	}
}

func TestProberRegistry_GetMissingProtocol(t *testing.T) {
	registry := NewProberRegistry()

	_, err := registry.Get(target.ProtocolICMP)
	if err == nil {
		t.Fatal("expected error for missing prober")
	}
}
