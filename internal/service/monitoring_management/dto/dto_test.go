package dto_test

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	dto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/service/monitoring_management/mappers"
	"WatchTower/internal/testutil"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func networkMappers() map[target.Protocol]dto.NetworkConfigMapper {
	return map[target.Protocol]dto.NetworkConfigMapper{
		target.ProtocolHTTP: mappers.HTTPNetworkConfigMapper{},
		target.ProtocolTCP:  mappers.TCPNetworkConfigMapper{},
		target.ProtocolICMP: mappers.ICMPNetworkConfigMapper{},
	}
}

func expectationMappers() map[target.Protocol]dto.ExpectationsMapper {
	return map[target.Protocol]dto.ExpectationsMapper{
		target.ProtocolHTTP: mappers.HTTPExpectationsMapper{},
		target.ProtocolTCP:  mappers.TCPExpectationsMapper{},
		target.ProtocolICMP: mappers.ICMPExpectationsMapper{},
	}
}

func TestMonitorDTOProtocols(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{ Protocol() target.Protocol }
		want   target.Protocol
		method string
	}{
		{name: "HTTP network config", value: dto.HTTPMonitorNetworkConfig{}, want: target.ProtocolHTTP, method: "HTTPMonitorNetworkConfig.Protocol"},
		{name: "TCP network config", value: dto.TCPMonitorNetworkConfig{}, want: target.ProtocolTCP, method: "TCPMonitorNetworkConfig.Protocol"},
		{name: "ICMP network config", value: dto.ICMPMonitorNetworkConfig{}, want: target.ProtocolICMP, method: "ICMPMonitorNetworkConfig.Protocol"},
		{name: "HTTP expectations", value: dto.HTTPMonitorExpectations{}, want: target.ProtocolHTTP, method: "HTTPMonitorExpectations.Protocol"},
		{name: "TCP expectations", value: dto.TCPMonitorExpectations{}, want: target.ProtocolTCP, method: "TCPMonitorExpectations.Protocol"},
		{name: "ICMP expectations", value: dto.ICMPMonitorExpectations{}, want: target.ProtocolICMP, method: "ICMPMonitorExpectations.Protocol"},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "monitoring/dto", tc.method, "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				if got := tc.value.Protocol(); got != tc.want {
					t.Fatalf("Protocol() = %q, want %q", got, tc.want)
				}
			})
		})
	}
}

func TestCreateMonitorDTOValidateProtocolConsistency(t *testing.T) {
	tests := []struct {
		name    string
		config  dto.MonitorNetworkConfig
		expect  dto.MonitorExpectations
		wantErr bool
	}{
		{name: "accepts matching protocols", config: dto.HTTPMonitorNetworkConfig{}, expect: dto.HTTPMonitorExpectations{}},
		{name: "rejects mismatched protocols", config: dto.HTTPMonitorNetworkConfig{}, expect: dto.TCPMonitorExpectations{}, wantErr: true},
		{name: "rejects missing network config", expect: dto.HTTPMonitorExpectations{}, wantErr: true},
		{name: "rejects missing expectations", config: dto.HTTPMonitorNetworkConfig{}, wantErr: true},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "monitoring/dto", "CreateMonitorDTO.ValidateProtocolConsistency", "decision-table", "classic", func(t *testing.T, _ *allure.Context) {
				err := (dto.CreateMonitorDTO{NetworkConfig: tc.config, Expectations: tc.expect}).ValidateProtocolConsistency()
				if (err != nil) != tc.wantErr {
					t.Fatalf("ValidateProtocolConsistency() error = %v", err)
				}
			})
		})
	}
}

func TestCreateMonitorDTOMapping(t *testing.T) {
	t.Run("maps network config", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "CreateMonitorDTO.ToDomainNetworkConfig", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			got, err := (dto.CreateMonitorDTO{NetworkConfig: dto.TCPMonitorNetworkConfig{Port: 443}}).ToDomainNetworkConfig(networkMappers())
			if err != nil || got != (target.TCPConfig{Port: 443}) {
				t.Fatalf("ToDomainNetworkConfig() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("rejects missing network config", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "CreateMonitorDTO.ToDomainNetworkConfig", "boundary", "classic", func(t *testing.T, _ *allure.Context) {
			if _, err := (dto.CreateMonitorDTO{}).ToDomainNetworkConfig(networkMappers()); err == nil {
				t.Fatal("expected missing network config error")
			}
		})
	})
	t.Run("maps expectations", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "CreateMonitorDTO.ToDomainExpectations", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			got, err := (dto.CreateMonitorDTO{Expectations: dto.TCPMonitorExpectations{MaxLatencyMs: 25}}).ToDomainExpectations(expectationMappers())
			if err != nil || got != (monitor.TCPExpectations{MaxLatencyMs: 25}) {
				t.Fatalf("ToDomainExpectations() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("rejects missing expectations", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "CreateMonitorDTO.ToDomainExpectations", "boundary", "classic", func(t *testing.T, _ *allure.Context) {
			if _, err := (dto.CreateMonitorDTO{}).ToDomainExpectations(expectationMappers()); err == nil {
				t.Fatal("expected missing expectations error")
			}
		})
	})
}

func TestUpdateMonitorDTOMapping(t *testing.T) {
	t.Run("maps present network config", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "UpdateMonitorDTO.ToDomainNetworkConfig", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			var config dto.MonitorNetworkConfig = dto.ICMPMonitorNetworkConfig{}
			got, err := (dto.UpdateMonitorDTO{NetworkConfig: &config}).ToDomainNetworkConfig(networkMappers())
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := got.(target.ICMPConfig); !ok {
				t.Fatalf("ToDomainNetworkConfig() = %#v", got)
			}
		})
	})
	t.Run("missing network config means no change", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "UpdateMonitorDTO.ToDomainNetworkConfig", "boundary", "classic", func(t *testing.T, _ *allure.Context) {
			got, err := (dto.UpdateMonitorDTO{}).ToDomainNetworkConfig(networkMappers())
			if err != nil || got != nil {
				t.Fatalf("ToDomainNetworkConfig() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("maps present expectations", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "UpdateMonitorDTO.ToDomainExpectations", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			var exp dto.MonitorExpectations = dto.ICMPMonitorExpectations{MaxLatencyMs: 12, MaxPacketLossPercent: 3}
			got, err := (dto.UpdateMonitorDTO{Expectations: &exp}).ToDomainExpectations(expectationMappers())
			if err != nil || got != (monitor.ICMPExpectations{MaxLatencyMs: 12, MaxPacketLossPercent: 3}) {
				t.Fatalf("ToDomainExpectations() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("missing expectations means no change", func(t *testing.T) {
		testutil.Case(t, "monitoring/dto", "UpdateMonitorDTO.ToDomainExpectations", "boundary", "classic", func(t *testing.T, _ *allure.Context) {
			got, err := (dto.UpdateMonitorDTO{}).ToDomainExpectations(expectationMappers())
			if err != nil || got != nil {
				t.Fatalf("ToDomainExpectations() = (%#v, %v)", got, err)
			}
		})
	})
}
