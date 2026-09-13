#!/usr/bin/env python3
"""Summarize Go/Allure results and display metrics computed by Cobertura.

This script neither instruments source code nor calculates source coverage.
"""
import argparse
import json
from pathlib import Path
import xml.etree.ElementTree as ET

parser = argparse.ArgumentParser()
parser.add_argument("run", type=Path)
parser.add_argument("--coverage", type=Path)
parser.add_argument("--exit-code", type=int, default=0)
args = parser.parse_args()
events = [json.loads(line) for line in (args.run / "tests.jsonl").read_text().splitlines() if line.strip()]
for event in events:
    if event.get("Action") == "fail" and "Test" not in event:
        print("FAIL", event.get("Package"))
        for item in events:
            if item.get("Package") == event.get("Package") and item.get("Output"):
                print(item["Output"], end="")
results = []
for path in sorted((args.run / "allure-results").glob("*-result.json")):
    data = json.loads(path.read_text())
    labels = {item["name"]: item["value"] for item in data.get("labels", [])}
    parameters = {item["name"]: item["value"] for item in data.get("parameters", [])}
    results.append({"name": data["name"], "fullName": data.get("fullName", ""),
                    "package": labels.get("package", "").removesuffix("_test"), "component": labels.get("suite", ""), "method": labels.get("subSuite", ""),
                    "technique": labels.get("technique", ""), "style": labels.get("style", ""),
                    "status": data.get("status"), "pid": parameters.get("pid", "")})
terminal = {(e["Package"], e["Test"]) for e in events if e.get("Test") and e.get("Action") in {"pass", "fail", "skip"}}
leaves = {(pkg, name) for pkg, name in terminal if not any(other_pkg == pkg and other.startswith(name + "/") for other_pkg, other in terminal)}
unreported = sorted((pkg, name) for pkg, name in leaves if not any(r["package"] == pkg and (name == r["name"] or name.startswith(r["name"] + "/")) for r in results))
processes = {}
for result in results:
    processes.setdefault(str(result["pid"]), set()).add(result["component"])
summary = {"exit_code": args.exit_code, "cases": len(results),
           "statuses": {status: sum(r["status"] == status for r in results)
                        for status in sorted({r["status"] for r in results})},
           "test_process_count": len(processes), "unreported_tests": unreported,
           "processes": {pid: sorted(components) for pid, components in sorted(processes.items())}}
if args.coverage:
    coverage = ET.parse(args.coverage).getroot()
    summary["line_coverage"] = {key: coverage.attrib[key] for key in ("line-rate", "lines-covered", "lines-valid")}
(args.run / "summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
(args.run / "cases.json").write_text(json.dumps(results, ensure_ascii=False, indent=2) + "\n")
print(json.dumps({k: v for k, v in summary.items() if k != "processes"}, ensure_ascii=False))

lines = ["# Матрица выполненных тестов", "", "Сгенерировано из метаданных Allure текущего прогона.", "", "| Компонент | Метод | Сценарий | Техника | Стиль | Результат |", "|---|---|---|---|---|---|"]
for result in sorted(results, key=lambda r: (r["package"], r["name"])):
    values = [result[key] for key in ("component", "method", "name", "technique", "style", "status")]
    lines.append("| " + " | ".join(str(value).replace("|", "\\|").replace("\n", " ") for value in values) + " |")
(args.run / "cases.md").write_text("\n".join(lines) + "\n")
if unreported:
    raise SystemExit("Some executed tests have no Allure result; see summary.json")
