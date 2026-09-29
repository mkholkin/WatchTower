#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
export LAB2_RUN_ID=${LAB2_RUN_ID:-$(date -u +%Y%m%dt%H%M%S)-$$-$RANDOM}
[[ "$LAB2_RUN_ID" =~ ^[a-z0-9][a-z0-9-]*$ ]] || { echo 'Invalid LAB2_RUN_ID: use lowercase letters, digits and hyphens' >&2; exit 2; }
export LAB2_OUTPUT="$PWD/build/lab2/runs/$LAB2_RUN_ID"
export LAB2_HISTORY_DIR="$PWD/build/lab2/history"
project="wt-lab2-$LAB2_RUN_ID"
if [[ ${1:-run} == build ]]; then
  export LAB2_TEST_IMAGE=${LAB2_TEST_IMAGE:-watchtower-lab2-tests:local}
else
  export LAB2_TEST_IMAGE=${LAB2_TEST_IMAGE:-$project-tests:local}
fi
compose=(docker compose -p "$project" -f tools/lab2/compose.yaml)
mkdir -p "$LAB2_HISTORY_DIR" "$(dirname "$LAB2_OUTPUT")"
if [[ ${1:-run} == build ]]; then
  "${compose[@]}" build postgres tests
  exit
fi
mkdir "$LAB2_OUTPUT"
runner_started=0
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
  if [[ "$runner_started" == 1 ]]; then
    # Wait for pipeline.py's finally block before removing PostgreSQL and Redis.
    docker stop --time 120 "$project-runner" > "$LAB2_OUTPUT/runner-stop.log" 2>&1 || true
    docker logs "$project-runner" > "$LAB2_OUTPUT/runner.log" 2>&1 || true
  fi
  if [[ "$status" != 0 && ! -f "$LAB2_OUTPUT/pipeline.json" ]]; then
    setup_failure
  fi
  "${compose[@]}" logs --no-color > "$LAB2_OUTPUT/compose.log" 2>&1 || true
  "${compose[@]}" down --volumes --remove-orphans > "$LAB2_OUTPUT/teardown.log" 2>&1 || status=1
  echo "Lab 2 artifacts: $LAB2_OUTPUT (exit $status)"
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
if [[ ${LAB2_SKIP_BUILD:-0} != 1 ]]; then
  "${compose[@]}" build postgres tests >> "$LAB2_OUTPUT/setup.log" 2>&1
fi
phase=image-inspection
docker image inspect "$LAB2_TEST_IMAGE" --format '{{.Id}}' > "$LAB2_OUTPUT/test-image.txt"
phase=readiness
"${compose[@]}" up -d --wait postgres redis probe >> "$LAB2_OUTPUT/setup.log" 2>&1
phase=runner
runner_started=1
"${compose[@]}" run -d --name "$project-runner" tests >> "$LAB2_OUTPUT/setup.log" 2>&1
# Bash processes traps immediately while waiting for a background command.
docker wait "$project-runner" > "$LAB2_OUTPUT/runner-exit.txt" &
wait "$!"
exit "$(cat "$LAB2_OUTPUT/runner-exit.txt")"
