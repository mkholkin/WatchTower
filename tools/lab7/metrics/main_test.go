package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fixture.go"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCyclomaticBoundary(t *testing.T) {
	for _, tc := range []struct {
		name           string
		cc, violations int
	}{{"cc10", 10, 0}, {"cc11", 11, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := analyze(fixture(t, tc.name), []string{"fixture.go"}, 10)
			if err != nil {
				t.Fatal(err)
			}
			if r.Summary.MaxCyclomatic != tc.cc || r.Summary.Violations != tc.violations {
				t.Fatalf("unexpected summary: %+v", r.Summary)
			}
		})
	}
}

func TestHalsteadHandCounts(t *testing.T) {
	r, err := analyze(fixture(t, "halstead"), []string{"fixture.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	// f has operators f, func, (), {}, return and operand 1, each once.
	want := halstead{5, 1, 5, 1, 6, 6, 6 * math.Log2(6), 2.5, 15 * math.Log2(6)}
	got := r.Functions[0].Halstead
	if got != want {
		t.Fatalf("f metrics: got %+v, want %+v", got, want)
	}
	// empty has operators empty, func, (), {} and no operands.
	want = halstead{4, 0, 4, 0, 4, 4, 8, 0, 0}
	if got := r.Functions[1].Halstead; got != want {
		t.Fatalf("empty metrics: got %+v, want %+v", got, want)
	}
}

func TestGeneratedExemption(t *testing.T) {
	r, err := analyze(fixture(t, "generated"), []string{"fixture.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary.Files != 1 || r.Summary.Functions != 0 || len(r.Summary.SkippedGenerated) != 1 {
		t.Fatalf("unexpected generated summary: %+v", r.Summary)
	}
}

func TestClosures(t *testing.T) {
	r, err := analyze(fixture(t, "closures"), []string{"fixture.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Functions) != 3 {
		t.Fatalf("got %d functions", len(r.Functions))
	}
	for i, want := range []int{3, 3, 2} {
		if got := r.Functions[i].Cyclomatic; got != want {
			t.Fatalf("function %d complexity: got %d, want %d", i, got, want)
		}
	}
	if r.Functions[1].ID == r.Functions[2].ID {
		t.Fatal("closures must have distinct identifiers")
	}
}

func TestParseErrorsFail(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.go"), []byte("package broken\nfunc ("), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := analyze(root, []string{"broken.go"}, 10); err == nil {
		t.Fatal("parse error accepted")
	}
}

func TestNonfiniteMetricsFail(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), -1} {
		if err := validate(halstead{Volume: value}); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}

func TestViolationReportIsWritten(t *testing.T) {
	r, err := analyze(fixture(t, "cc11"), []string{"fixture.go"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "metrics.json")
	if err := writeReport(r, output, 10); err == nil {
		t.Fatal("complexity violation accepted")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.Violations != 1 {
		t.Fatalf("unexpected report: %+v", got.Summary)
	}
}

func TestReportWriteFailure(t *testing.T) {
	if err := writeReport(report{}, filepath.Join(t.TempDir(), "missing", "metrics.json"), 10); err == nil {
		t.Fatal("write failure accepted")
	}
}
