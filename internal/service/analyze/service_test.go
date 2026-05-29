package analyzation_service

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probeRepo.EXPECT().FetchUnprocessed(gomock.Any(), 10).DoAndReturn(
		func(context.Context, int) ([]*probe.Result, error) {
			cancel()
			return []*probe.Result{}, nil
		},
	)

	if err := svc.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
