package mappers

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/testutil"
	"reflect"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestNetworkConfigMappers(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		mapFunc func(dto.MonitorNetworkConfig) (target.NetworkConfig, error)
		input   dto.MonitorNetworkConfig
		want    target.NetworkConfig
		bad     dto.MonitorNetworkConfig
	}{
		{name: "HTTP", method: "HTTPNetworkConfigMapper.Map", mapFunc: (HTTPNetworkConfigMapper{}).Map, input: dto.HTTPMonitorNetworkConfig{Method: "POST", Headers: map[string]string{"X-Test": "yes"}, Body: "ok", FollowRedirects: true}, want: target.HTTPConfig{Method: "POST", Headers: map[string]string{"X-Test": "yes"}, Body: "ok", FollowRedirects: true}, bad: dto.TCPMonitorNetworkConfig{Port: 80}},
		{name: "TCP", method: "TCPNetworkConfigMapper.Map", mapFunc: (TCPNetworkConfigMapper{}).Map, input: dto.TCPMonitorNetworkConfig{Port: 443}, want: target.TCPConfig{Port: 443}, bad: dto.ICMPMonitorNetworkConfig{}},
		{name: "ICMP", method: "ICMPNetworkConfigMapper.Map", mapFunc: (ICMPNetworkConfigMapper{}).Map, input: dto.ICMPMonitorNetworkConfig{}, want: target.ICMPConfig{}, bad: dto.HTTPMonitorNetworkConfig{}},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name+" maps typed input", func(t *testing.T) {
			testutil.Case(t, "monitoring/mappers", tc.method, "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				got, err := tc.mapFunc(tc.input)
				if err != nil || !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("Map() = (%#v, %v), want %#v", got, err, tc.want)
				}
			})
		})
		t.Run(tc.name+" rejects another protocol type", func(t *testing.T) {
			testutil.Case(t, "monitoring/mappers", tc.method, "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				if _, err := tc.mapFunc(tc.bad); err == nil {
					t.Fatal("Map() accepted wrong protocol DTO")
				}
			})
		})
	}
}

func TestExpectationMappers(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		mapFunc func(dto.MonitorExpectations) (monitor.Expectations, error)
		input   dto.MonitorExpectations
		want    monitor.Expectations
		bad     dto.MonitorExpectations
	}{
		{name: "HTTP", method: "HTTPExpectationsMapper.Map", mapFunc: (HTTPExpectationsMapper{}).Map, input: dto.HTTPMonitorExpectations{StatusCodes: []int{200, 204}, MaxLatencyMs: 50}, want: monitor.HTTPExpectations{StatusCodes: []int{200, 204}, MaxLatencyMs: 50}, bad: dto.TCPMonitorExpectations{}},
		{name: "TCP", method: "TCPExpectationsMapper.Map", mapFunc: (TCPExpectationsMapper{}).Map, input: dto.TCPMonitorExpectations{MaxLatencyMs: 60}, want: monitor.TCPExpectations{MaxLatencyMs: 60}, bad: dto.ICMPMonitorExpectations{}},
		{name: "ICMP", method: "ICMPExpectationsMapper.Map", mapFunc: (ICMPExpectationsMapper{}).Map, input: dto.ICMPMonitorExpectations{MaxLatencyMs: 70, MaxPacketLossPercent: 5}, want: monitor.ICMPExpectations{MaxLatencyMs: 70, MaxPacketLossPercent: 5}, bad: dto.HTTPMonitorExpectations{}},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name+" maps typed input", func(t *testing.T) {
			testutil.Case(t, "monitoring/mappers", tc.method, "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				got, err := tc.mapFunc(tc.input)
				if err != nil || !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("Map() = (%#v, %v), want %#v", got, err, tc.want)
				}
			})
		})
		t.Run(tc.name+" rejects another protocol type", func(t *testing.T) {
			testutil.Case(t, "monitoring/mappers", tc.method, "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				if _, err := tc.mapFunc(tc.bad); err == nil {
					t.Fatal("Map() accepted wrong protocol DTO")
				}
			})
		})
	}
}
