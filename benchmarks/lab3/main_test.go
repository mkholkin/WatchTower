package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestLibrariesProduceEquivalentMonitorChecks(t *testing.T) {
	for _, size := range []int{1, 32} {
		data := fixture(size)
		want, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, lib := range libraries {
			for _, scenario := range []string{"encode", "decode", "roundtrip"} {
				op, err := prepare(lib, scenario, data)
				if err != nil {
					t.Fatal(err)
				}
				if err := op(); err != nil {
					t.Fatal(err)
				}
				if scenario != "encode" && !reflect.DeepEqual(sinkChecks, data) {
					t.Fatalf("%s/%s: decoded data differs", lib.name, scenario)
				}
				if scenario != "decode" && string(sinkBytes) != string(want) {
					t.Fatalf("%s/%s: JSON differs", lib.name, scenario)
				}
			}
		}
	}
}

func TestTrialSeparatesPhasesAndCountsAllOperations(t *testing.T) {
	cfg := config{Warmup: 2, Batches: 3, BatchSize: 4, LatencySamples: 5, Teardown: 2}
	result, err := runTrial(7, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 12 {
		t.Fatalf("cases=%d", len(result.Cases))
	}
	seen := map[string]bool{}
	for _, c := range result.Cases {
		if seen[c.ID] {
			t.Fatal("duplicate case", c.ID)
		}
		seen[c.ID] = true
		if c.Warmup.Operations != 2 || c.Measurement.Operations != 12 || c.Latency.Operations != 5 || c.Teardown.Operations != 2 || c.TotalOperations != 21 {
			t.Fatalf("wrong accounting: %+v", c)
		}
		if len(c.Batches) != 3 || len(c.LatenciesNS) != 5 || c.DurationNS <= 0 || c.Measurement.ElapsedNS <= 0 {
			t.Fatalf("missing timings: %+v", c)
		}
		for _, b := range c.Batches {
			if b.NSPerOp <= 0 || b.OffsetNS < 0 {
				t.Fatal("invalid batch", b)
			}
		}
	}
}
