#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
root=$PWD
output="$root/build/lab1"
bin="$root/build/tools"
mkdir -p "$output" "$bin"
packages=()
while IFS= read -r package; do [[ -n "$package" ]] && packages+=("$package"); done < tools/lab1/packages.txt
mode=${1:-unit}
export GOTOOLCHAIN=local
export GOFLAGS="${GOFLAGS:-} -mod=readonly"
if [[ "$mode" != prepare ]]; then
  export GOPROXY=off GOSUMDB=off GOTELEMETRY=off
fi
# Keep the original test failure even if subsequent reporting also fails.
record_failure() { if [[ "$status" == 0 ]]; then status=$1; fi; }

case "$mode" in
prepare)
  go mod download
  GOBIN="$bin" go install github.com/rillig/gobco@v1.3.4
  GOBIN="$bin" go install github.com/boumenot/gocover-cobertura@v1.5.0
  npm ci --prefix tools/lab1 --no-audit --no-fund
  ;;
unit|shuffle|race|coverage|processes)
  run=${LAB1_RUN:-latest}
  args=(-count=1 -timeout=120s -json)
  case "$mode" in
  shuffle)
    export LAB1_SEED=${SEED:-42}
    [[ "$LAB1_SEED" =~ ^[0-9]+$ ]] || { echo 'SEED must be a nonnegative integer' >&2; exit 2; }
    args+=("-shuffle=$LAB1_SEED"); run="shuffle-$LAB1_SEED" ;;
  race) args+=(-race); run=race ;;
  coverage)
    coverpkg=$(IFS=,; echo "${packages[*]}")
    rm -f "$output/coverage.out"
    args+=(-covermode=atomic "-coverpkg=$coverpkg" "-coverprofile=$output/coverage.out") ;;
  processes) args+=(-p=1); run=processes-serial ;;
  esac
  dest="$output/runs/$run"
  rm -rf "$dest"
  mkdir -p "$dest/allure-results"
  export ALLURE_RESULTS_DIR="$dest/allure-results"
  status=0
  go test "${args[@]}" "${packages[@]}" > "$dest/tests.jsonl" 2> "$dest/stderr.log" || status=$?
  cat "$dest/stderr.log"
  python3 tools/lab1/summarize.py "$dest" --exit-code "$status" || record_failure "$?"
  if [[ "$mode" == coverage && -f "$output/coverage.out" ]]; then
    go tool cover -func="$output/coverage.out" > "$output/statements.txt" || record_failure "$?"
    go tool cover -html="$output/coverage.out" -o "$output/coverage.html" || record_failure "$?"
    "$bin/gocover-cobertura" -ignore-gen-files -ignore-files '(internal/service/testmocks/|internal/testutil/|sqlcgen/|internal/infra/probe/(tcp|icmp)\.go)' < "$output/coverage.out" > "$output/coverage.xml" || record_failure "$?"
    python3 tools/lab1/summarize.py "$dest" --coverage "$output/coverage.xml" --exit-code "$status" || record_failure "$?"
  fi
  # Even a failed test run produces a report; never mask the test exit code.
  if [[ "$status" != 0 ]]; then
    LAB1_RUN="$run" "$0" report || true
  fi
  exit "$status"
  ;;
branch)
  dest="$output/branches"
  rm -rf "$dest"
  mkdir -p "$dest/allure-results"
  export ALLURE_RESULTS_DIR="$dest/allure-results"
  snapshot=$(mktemp -d)
  trap 'rm -rf "$snapshot"' EXIT
  tar --exclude=.git --exclude=build --exclude=.cache --exclude=tools --exclude=watchtower-ui --exclude=tiopo --exclude=img --exclude=docs --exclude=allure-results -cf - . | tar -xf - -C "$snapshot"
  status=0
  for package in "${packages[@]}"; do
    name=${package#./}; name=${name//\//_}
    (cd "$snapshot"; "$bin/gobco" -branch -test=-count=1 -test=-timeout=120s -stats "$dest/$name.json" "$package") > "$dest/$name.log" 2>&1 || record_failure "$?"
    cat "$dest/$name.log"
  done
  exit "$status"
  ;;
report)
  tools/lab1/node_modules/.bin/allure generate "$output/runs/${LAB1_RUN:-latest}/allure-results" --config "$root/tools/lab1/allurerc.mjs" --output "$output/allure-report"
  ;;
open)
  tools/lab1/node_modules/.bin/allure open "$output/allure-report"
  ;;
all)
  status=0
  "$0" coverage || record_failure "$?"
  "$0" branch || record_failure "$?"
  "$0" report || record_failure "$?"
  exit "$status"
  ;;
*) echo "unknown mode: $mode" >&2; exit 2 ;;
esac
