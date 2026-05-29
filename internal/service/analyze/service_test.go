package service

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestProbeAnalyzationService_Run_NoResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitorRepo := testmocks.NewMockMonitorRepository(ctrl)
	probeRepo := testmocks.NewMockProbeResultRepository(ctrl)
	summaryRepo := testmocks.NewMockProbeSummaryRepository(ctrl)
	evaluator := testmocks.NewMockProbeEvaluator(ctrl)

	logger := testutil.NoopLogger()
	svc := NewProbeAnalyzationService(
		monitorRepo,
		probeRepo,
		summaryRepo,
		evaluator,
		nil,
		ProbeAnalyzationServiceConfig{FetchLimit: 10, LoadSheddingThreshold: 5},
		logger,
	)

	probeRepo.EXPECT().FetchUnprocessed(gomock.Any(), 10).Return([]probe.Result{}, nil)

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
