COMMON ?= 5000000
STACKS := go php linux
TIME := /usr/bin/time -f "  %es  %MKB peak"

.PHONY: data build run verify bench clean

data: ## generate data/old.csv + data/new.csv (6M rows each by default)
	go run ./gen -common $(COMMON)

build:
	go build -o bin/diff-go ./go

run: build ## run all three stacks once
	./bin/diff-go
	php -d memory_limit=-1 php/diff.php
	bash linux/diff.sh

verify: ## every stack must match the expected counts and each other
	@for s in $(STACKS); do \
	  for f in new removed modified; do sort output/$$s/$$f-products.csv > /tmp/ds-$$s-$$f; done; \
	done
	@for s in php linux; do for f in new removed modified; do \
	  cmp -s /tmp/ds-go-$$f /tmp/ds-$$s-$$f || { echo "FAIL: $$s $$f differs from go"; exit 1; }; \
	done; done
	@got="New: $$(($$(wc -l < output/go/new-products.csv)-1))  Removed: $$(($$(wc -l < output/go/removed-products.csv)-1))  Modified: $$(($$(wc -l < output/go/modified-products.csv)-1))"; \
	  test "$$got" = "$$(cat data/expected.txt)" || { echo "FAIL: got $$got, expected $$(cat data/expected.txt)"; exit 1; }; \
	  echo "OK: all stacks identical - $$got"
	@rm -f /tmp/ds-*

bench: build ## time + peak memory per stack
	@echo "go:";    $(TIME) ./bin/diff-go
	@echo "php:";   $(TIME) php -d memory_limit=-1 php/diff.php
	@echo "linux:"; $(TIME) bash linux/diff.sh

clean:
	rm -rf bin output
