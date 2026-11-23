package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fzipp/gocyclo"
	"github.com/yagipy/maintidx/pkg/halstvol"
)

type halstead struct {
	DistinctOperators int     `json:"n1"`
	DistinctOperands  int     `json:"n2"`
	TotalOperators    int     `json:"N1"`
	TotalOperands     int     `json:"N2"`
	Vocabulary        int     `json:"vocabulary"`
	Length            int     `json:"length"`
	Volume            float64 `json:"volume"`
	Difficulty        float64 `json:"difficulty"`
	Effort            float64 `json:"effort"`
}

type function struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Cyclomatic int      `json:"cyclomatic"`
	Halstead   halstead `json:"halstead"`
}

type summary struct {
	Files            int      `json:"files"`
	Functions        int      `json:"functions"`
	SkippedGenerated []string `json:"skipped_generated"`
	MaxCyclomatic    int      `json:"max_cyclomatic"`
	Violations       int      `json:"violations"`
}

type report struct {
	Summary   summary    `json:"summary"`
	Functions []function `json:"functions"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", "metrics.json", "JSON report path")
	limit := flag.Int("max-cyclomatic", 10, "maximum accepted function complexity")
	flag.Parse()
	if err := run(*root, *output, *limit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, output string, limit int) error {
	if limit < 1 {
		return errors.New("max-cyclomatic must be positive")
	}
	files, err := sourceFiles(root)
	if err != nil {
		return err
	}
	result, err := analyze(root, files, limit)
	if err != nil {
		return err
	}
	return writeReport(result, output, limit)
}

func writeReport(result report, output string, limit int) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if result.Summary.Violations != 0 {
		return fmt.Errorf("%d functions exceed cyclomatic complexity %d (report: %s)", result.Summary.Violations, limit, output)
	}
	return nil
}

func sourceFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", "*.go")
	cmd.Dir = root
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list Go sources: %w", err)
	}
	files := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
	if len(data) == 0 {
		return nil, errors.New("no Go sources found")
	}
	return files, nil
}

func analyze(root string, files []string, limit int) (report, error) {
	result := report{Summary: summary{SkippedGenerated: []string{}}, Functions: []function{}}
	for _, name := range files {
		fs := token.NewFileSet()
		file, err := parser.ParseFile(fs, filepath.Join(root, name), nil, parser.ParseComments)
		if err != nil {
			return result, fmt.Errorf("parse %s: %w", name, err)
		}
		result.Summary.Files++
		if ast.IsGenerated(file) {
			result.Summary.SkippedGenerated = append(result.Summary.SkippedGenerated, name)
			continue
		}
		functions, err := analyzeFunctions(fs, file, name)
		if err != nil {
			return result, err
		}
		for _, fn := range functions {
			result.add(fn, limit)
		}
	}
	return result, nil
}

func (r *report) add(fn function, limit int) {
	r.Functions = append(r.Functions, fn)
	r.Summary.Functions++
	r.Summary.MaxCyclomatic = max(r.Summary.MaxCyclomatic, fn.Cyclomatic)
	if fn.Cyclomatic > limit {
		r.Summary.Violations++
	}
}

func analyzeFunctions(fs *token.FileSet, file *ast.File, path string) ([]function, error) {
	functions := []function{}
	var failure error
	ast.Inspect(file, func(node ast.Node) bool {
		name, ok := functionName(node)
		if !ok {
			return true
		}
		metrics, err := measure(node)
		if err != nil {
			failure = fmt.Errorf("%s:%d: %w", path, fs.Position(node.Pos()).Line, err)
			return false
		}
		functions = append(functions, function{ID: fmt.Sprintf("%s@%s:%d:%d", name, path, fs.Position(node.Pos()).Line, fs.Position(node.Pos()).Column), Name: name, File: path, Line: fs.Position(node.Pos()).Line, Cyclomatic: gocyclo.Complexity(node), Halstead: metrics})
		return true
	})
	return functions, failure
}

func functionName(node ast.Node) (string, bool) {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		return fn.Name.Name, true
	case *ast.FuncLit:
		return "func literal", true
	default:
		return "", false
	}
}

func measure(node ast.Node) (halstead, error) {
	h := halstvol.HalstVol{Coef: halstvol.Coef{Opt: map[string]int{}, Opd: map[string]int{}}}
	ast.Inspect(node, func(n ast.Node) bool { h.Analyze(n); return true })
	result := halstead{DistinctOperators: len(h.Coef.Opt), DistinctOperands: len(h.Coef.Opd)}
	for _, n := range h.Coef.Opt {
		result.TotalOperators += n
	}
	for _, n := range h.Coef.Opd {
		result.TotalOperands += n
	}
	result.Vocabulary = result.DistinctOperators + result.DistinctOperands
	result.Length = result.TotalOperators + result.TotalOperands
	if result.Vocabulary > 0 {
		h.Calc()
		result.Volume = h.Val
	}
	if result.DistinctOperands > 0 {
		result.Difficulty = float64(result.DistinctOperators) / 2 * float64(result.TotalOperands) / float64(result.DistinctOperands)
	}
	result.Effort = result.Volume * result.Difficulty
	return result, validate(result)
}

func validate(h halstead) error {
	for _, value := range []float64{h.Volume, h.Difficulty, h.Effort} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("invalid Halstead metrics")
		}
	}
	return nil
}
