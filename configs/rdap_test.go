package configs

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestRDAPConfigYAML(t *testing.T) {
	var config Config
	err := yaml.Unmarshal([]byte("rdap:\n  enabled: true\n  base_url: http://mock:8080/com/v1\n  timeout_ms: 1200\n"), &config)
	if err != nil {
		t.Fatal(err)
	}
	if !config.RDAP.Enabled || config.RDAP.BaseURL != "http://mock:8080/com/v1" || config.RDAP.TimeoutMS != 1200 {
		t.Fatalf("unexpected config %+v", config.RDAP)
	}
}
