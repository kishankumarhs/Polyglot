.PHONY: codegen test build-native demo clean doctor check-submodules

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

clean:
	rm -rf dist build
	rm -rf bindings/python/polyglot_logger/native
	rm -rf bindings/node/native bindings/node/dist
	rm -rf bindings/dotnet/Polyglot.Logger/native
	rm -f codegen liblogger.h
