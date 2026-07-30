.PHONY: codegen test build-native demo clean doctor check-submodules bench bench-smoke bench-pprof bench-charts

codegen:
	go run ./cmd/codegen

test:
	go test ./internal/logger -count=1

test-race:
	CGO_ENABLED=1 go test ./internal/logger -race -count=1

build-native: codegen
	bash scripts/build-native.sh dist

demo:
	go run ./cmd/logger-demo

doctor:
	go run ./cmd/polyglot doctor

check-submodules:
	bash scripts/check-submodules.sh

bench:
	bash scripts/bench.sh

bench-smoke:
	bash scripts/bench-smoke.sh

bench-pprof:
	bash scripts/bench-pprof.sh

bench-charts:
	python scripts/bench-summarize.py || true
	python scripts/bench-charts.py

clean:
	rm -rf dist build
	rm -rf bindings/python/polyglot_logger/native
	rm -rf bindings/node/native bindings/node/dist
	rm -rf bindings/dotnet/Polyglot.Logger/native
	rm -f codegen liblogger.h
	rm -rf bench/results/*.txt bench/results/*.pprof bench/results/*.csv bench/results/latest.md
