#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
mode=${1:-run}
case "$mode" in
  run|build|prepare|unit|integration|e2e|finish) ;;
  *) echo 'Usage: run.sh [run|build|prepare|unit|integration|e2e|finish]' >&2; exit 2 ;;
esac
if [[ "$mode" != run && "$mode" != build && -z ${LAB2_RUN_ID:-} ]]; then
  echo 'Separate CI steps require a shared LAB2_RUN_ID' >&2
  exit 2
fi
export LAB2_RUN_ID=${LAB2_RUN_ID:-$(date -u +%Y%m%dt%H%M%S)-$$-$RANDOM}
[[ "$LAB2_RUN_ID" =~ ^[a-z0-9][a-z0-9-]*$ ]] || { echo 'Invalid LAB2_RUN_ID: use lowercase letters, digits and hyphens' >&2; exit 2; }
export LAB2_OUTPUT="$PWD/build/lab2/runs/$LAB2_RUN_ID"
export LAB2_HISTORY_DIR="$PWD/build/lab2/history"
project="wt-lab2-$LAB2_RUN_ID"
if [[ "$mode" == build ]]; then
  export LAB2_TEST_IMAGE=${LAB2_TEST_IMAGE:-watchtower-lab2-tests:local}
else
  export LAB2_TEST_IMAGE=${LAB2_TEST_IMAGE:-$project-tests:local}
fi
compose=(docker compose -p "$project" -f tools/lab2/compose.yaml)
mkdir -p "$LAB2_HISTORY_DIR" "$(dirname "$LAB2_OUTPUT")"
if [[ "$mode" == build ]]; then
  "${compose[@]}" build postgres tests
  exit
fi
if [[ "$mode" == run || "$mode" == prepare ]]; then
  mkdir "$LAB2_OUTPUT"
elif [[ "$mode" == finish ]]; then
  mkdir -p "$LAB2_OUTPUT"
elif [[ ! -f "$LAB2_OUTPUT/prepared" ]]; then
  echo 'Run prepare before starting a separate test step' >&2
  exit 2
fi
runner="$project-runner"
runner_log="$LAB2_OUTPUT/runner.log"
if [[ "$mode" != run && "$mode" != prepare ]]; then
  runner="$runner-$mode"
  runner_log="$LAB2_OUTPUT/runner-$mode.log"
fi
runner_started=0
logs_pid=
phase=build
setup_failure() {
  printf 'Lab 2 preparation failed during %s (exit %s). See setup.log, runner.log and compose.log.\n' "$phase" "$status" > "$LAB2_OUTPUT/setup-failure.txt"
  for image in "$LAB2_TEST_IMAGE" watchtower-lab2-tests:local; do
    if docker image inspect "$image" >/dev/null 2>&1; then
      if docker run --rm --init --pull=never --network=none --entrypoint python3 \
        -e "LAB2_RUN_ID=$LAB2_RUN_ID" \
        -v "$LAB2_OUTPUT:/artifacts" -v "$LAB2_HISTORY_DIR:/history" \
        -v "$PWD/tools/lab2/setup_failure.py:/tmp/lab2-setup-failure.py:ro" \
        "$image" /tmp/lab2-setup-failure.py > "$LAB2_OUTPUT/setup-report.log" 2>&1; then
        return
      fi
    fi
  done
  printf 'Container report unavailable; trying host Python and already-installed Allure.\n' >> "$LAB2_OUTPUT/setup-failure.txt"
  if command -v python3 >/dev/null; then
    python3 tools/lab2/setup_failure.py --artifacts "$LAB2_OUTPUT" --history "$LAB2_HISTORY_DIR" \
      >> "$LAB2_OUTPUT/setup-report.log" 2>&1 || true
  else
    printf 'No Python or usable test image: Allure results/report could not be generated.\n' >> "$LAB2_OUTPUT/setup-failure.txt"
  fi
}
cleanup() {
  status=$?
  trap - EXIT
  trap '' INT TERM
  if [[ "$mode" == prepare && "$status" == 0 ]]; then
    echo "Lab 2 environment ready: $project"
    exit 0
  fi
  if [[ "$runner_started" == 1 ]]; then
    # Wait for pipeline.py's finally block before removing PostgreSQL and Redis.
    docker stop --time 120 "$runner" >> "$LAB2_OUTPUT/runner-stop.log" 2>&1 || true
    if [[ -n "$logs_pid" ]]; then
      kill "$logs_pid" 2>/dev/null || true
      wait "$logs_pid" 2>/dev/null || true
    fi
    docker logs "$runner" > "$runner_log" 2>&1 || true
    if [[ "$mode" != run ]]; then
      cat "$runner_log" >> "$LAB2_OUTPUT/runner.log"
    fi
  fi
  if [[ "$status" != 0 && ( "$mode" == unit || "$mode" == integration || "$mode" == e2e ) ]]; then
    printf '%s\n' "$mode" > "$LAB2_OUTPUT/failed-stage.txt"
  fi
  if [[ "$status" != 0 && ( "$mode" == run || "$mode" == prepare ) && ! -f "$LAB2_OUTPUT/pipeline.json" ]]; then
    setup_failure
  fi
  if [[ "$mode" == run || "$mode" == finish || "$mode" == prepare ]]; then
    "${compose[@]}" logs --no-color > "$LAB2_OUTPUT/compose.log" 2>&1 || true
    "${compose[@]}" down --volumes --remove-orphans > "$LAB2_OUTPUT/teardown.log" 2>&1 || status=1
  fi
  echo "Lab 2 artifacts: $LAB2_OUTPUT (exit $status)"
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
if [[ "$mode" == run || "$mode" == prepare ]]; then
  if [[ ${LAB2_SKIP_BUILD:-0} != 1 ]]; then
    "${compose[@]}" build postgres tests 2>&1 | tee -a "$LAB2_OUTPUT/setup.log"
  fi
  phase=image-inspection
  docker image inspect "$LAB2_TEST_IMAGE" --format '{{.Id}}' > "$LAB2_OUTPUT/test-image.txt"
  phase=readiness
  "${compose[@]}" up -d --wait postgres redis probe 2>&1 | tee -a "$LAB2_OUTPUT/setup.log"
  touch "$LAB2_OUTPUT/prepared"
  if [[ "$mode" == prepare ]]; then
    exit 0
  fi
elif [[ "$mode" == finish && ( ! -f "$LAB2_OUTPUT/prepared" || -f "$LAB2_OUTPUT/setup-failure.txt" ) ]]; then
  # Preparation already emitted its report and removed any partially started stores.
  exit 0
fi
phase=runner
runner_started=1
if [[ "$mode" == run ]]; then
  "${compose[@]}" run -d --name "$runner" tests 2>&1 | tee -a "$LAB2_OUTPUT/setup.log"
else
  "${compose[@]}" run -d --name "$runner" tests "$mode" 2>&1 | tee -a "$LAB2_OUTPUT/setup.log"
fi
docker logs --follow "$runner" &
logs_pid=$!
# Bash processes traps immediately while waiting for a background command.
docker wait "$runner" > "$LAB2_OUTPUT/runner-exit.txt" &
wait "$!"
# Drain the final log lines before cleanup stops the follower.
wait "$logs_pid" || true
logs_pid=
exit "$(cat "$LAB2_OUTPUT/runner-exit.txt")"
