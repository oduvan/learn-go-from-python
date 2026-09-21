# Runnable checks for the third-party examples in docs/13-third-party-libraries/.
#
#   make verify    bring services up, run everything, tear them down
#   make test      run the Go tests against whatever is already running
#   make up/down   manage the Postgres and Valkey containers

SHELL := /usr/bin/env bash

.PHONY: verify test up down generate lint-fixture clean

verify: up
	@$(MAKE) --no-print-directory test; status=$$?; \
	 $(MAKE) --no-print-directory down; exit $$status

test: generate
	go test ./...
	@./lint_test.sh

# Tests skip when a service is missing. In CI set EXAMPLES_REQUIRE_INFRA=1
# so a missing service fails loudly instead of quietly passing.
ci: generate
	EXAMPLES_REQUIRE_INFRA=1 go test ./...
	@./lint_test.sh

up:
	docker compose up -d --wait

down:
	docker compose down -v

generate:
	@command -v templ >/dev/null 2>&1 && templ generate ./... >/dev/null \
	  || echo "note: templ CLI not found, using the committed page_templ.go"

clean:
	go clean -testcache
