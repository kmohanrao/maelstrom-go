# Go build configurations
GO := go
BINARY_DIR := bin
SOURCE_DIR := cmd

# Maelstrom configurations
MAELSTROM := ./maelstrom/maelstrom
NODE_COUNT := 5
HIGH_NODE_COUNT := 25
LOW_TIME_LIMIT := 20
TIME_LIMIT := 30
RATE := 10
HIGHER_RATE := 100
LATENCY := 100
TOPOLOGY := tree4

# Challenge names
CHALLENGES := echo \
	broadcast \
	unique-ids \
	g-counter \
	pn-counter \
	txn-list-append \
	kafka

# Create binary paths
BINARIES := $(addprefix $(BINARY_DIR)/, $(CHALLENGES))

# Build targets
.PHONY: all
all: build test-all

# Create binary directory
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# Generic build rule for all challenges
$(BINARY_DIR)/%: $(SOURCE_DIR)/%/main.go | $(BINARY_DIR)
	$(GO) build -o $@ $<

# Build all binaries
.PHONY: build
build: $(BINARIES)

# Clean built binaries and test results
.PHONY: clean
clean:
	rm -rf $(BINARY_DIR)
	rm -rf store/

# Development tools
.PHONY: deps fmt lint test
deps:
	$(GO) mod download
	$(GO) mod verify

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

# Maelstrom serve for debugging
.PHONY: serve
serve:
	$(MAELSTROM) serve

# Custom serve target for specific test with history
.PHONY: serve-history
serve-history:
	@if [ -z "$(history)" ]; then \
		echo "Please specify a history file with history=<path>"; \
		echo "Example: make serve-history history=store/echo/latest/history.txt"; \
		exit 1; \
	fi
	$(MAELSTROM) serve $(history)

# Maelstrom test targets
.PHONY: test-all
test-all: test-echo test-broadcast test-unique-ids test-g-counter test-pn-counter test-txn-list-append test-kafka

# Echo test
.PHONY: test-echo
test-echo: $(BINARY_DIR)/echo
	$(MAELSTROM) test \
		-w echo \
		--bin $(BINARY_DIR)/echo \
		--node-count 1 \
		--time-limit 10

# Broadcast tests
.PHONY: test-broadcast
test-broadcast: $(BINARY_DIR)/broadcast test-broadcast-single test-broadcast-multi test-broadcast-fault test-broadcast-stable-latency test-broadcast-efficiency

test-broadcast-single: $(BINARY_DIR)/broadcast
	$(MAELSTROM) test \
		-w broadcast \
		--bin $(BINARY_DIR)/broadcast \
		--node-count 1 \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE)

test-broadcast-multi: $(BINARY_DIR)/broadcast
	$(MAELSTROM) test \
		-w broadcast \
		--bin $(BINARY_DIR)/broadcast \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE)

test-broadcast-fault: $(BINARY_DIR)/broadcast
	$(MAELSTROM) test \
		-w broadcast \
		--bin $(BINARY_DIR)/broadcast \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis partition \
		--topology $(TOPOLOGY)

test-broadcast-stable-latency: $(BINARY_DIR)/broadcast
	$(MAELSTROM) test \
		-w broadcast \
		--bin $(BINARY_DIR)/broadcast \
		--node-count $(HIGH_NODE_COUNT) \
		--time-limit $(LOW_TIME_LIMIT) \
		--rate $(HIGHER_RATE) \
		--latency $(LATENCY) \
		--topology $(TOPOLOGY)

test-broadcast-efficiency: $(BINARY_DIR)/broadcast
	$(MAELSTROM) test \
		-w broadcast \
		--bin $(BINARY_DIR)/broadcast \
		--node-count $(HIGH_NODE_COUNT) \
		--time-limit $(LOW_TIME_LIMIT) \
		--rate $(HIGHER_RATE) \
		--latency $(LATENCY)

# Unique IDs test
.PHONY: test-unique-ids
test-unique-ids: $(BINARY_DIR)/unique-ids
	$(MAELSTROM) test \
		-w unique-ids \
		--bin $(BINARY_DIR)/unique-ids \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--node-count $(NODE_COUNT)

# Grow-only counter test
.PHONY: test-g-counter
test-g-counter: $(BINARY_DIR)/g-counter
	$(MAELSTROM) test \
		-w g-counter \
		--bin $(BINARY_DIR)/g-counter \
		--node-count $(NODE_COUNT) \
		--rate $(RATE) \
		--time-limit $(TIME_LIMIT) \
		--nemesis partition

# Positive-negative counter test
.PHONY: test-pn-counter
test-pn-counter: $(BINARY_DIR)/pn-counter
	$(MAELSTROM) test \
		-w pn-counter \
		--bin $(BINARY_DIR)/pn-counter \
		--node-count $(NODE_COUNT) \
		--rate $(RATE) \
		--time-limit $(TIME_LIMIT) \
		--nemesis partition

# Transactional list append test
.PHONY: test-txn-list-append
test-txn-list-append: $(BINARY_DIR)/txn-list-append
	$(MAELSTROM) test \
		-w txn-list-append \
		--bin $(BINARY_DIR)/txn-list-append \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis partition

# Kafka-style log test
.PHONY: test-kafka
test-kafka: $(BINARY_DIR)/kafka
	$(MAELSTROM) test \
		-w kafka \
		--bin $(BINARY_DIR)/kafka \
		--node-count $(NODE_COUNT) \
		--concurrency 2n \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build        - Build all challenge binaries"
	@echo "  clean        - Remove built binaries and test results"
	@echo "  deps         - Download and verify dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  test         - Run Go tests"
	@echo ""
	@echo "Debug targets:"
	@echo "  serve        - Start Maelstrom web server for visualization"
	@echo "  serve-history history=<path> - Visualize a specific test history"
	@echo ""
	@echo "Maelstrom test targets:"
	@echo "  test-all     - Run all Maelstrom tests"
	@echo "  test-echo    - Run echo test"
	@echo "  test-broadcast                  - Run all broadcast tests"
	@echo "  test-broadcast-single           - Run single-node broadcast test"
	@echo "  test-broadcast-multi            - Run multi-node broadcast test"
	@echo "  test-broadcast-fault            - Run broadcast test with network partitions"
	@echo "  test-broadcast-stable-latency   - Run broadcast test with latency"
	@echo "  test-broadcast-efficiency       - Run broadcast test with latency for efficiency"
	@echo "  test-unique-ids                 - Run unique IDs test"
	@echo "  test-g-counter                  - Run grow-only counter test"
	@echo "  test-pn-counter                 - Run positive-negative counter test"
	@echo "  test-txn-list-append            - Run transactional list append test"
	@echo "  test-kafka                      - Run Kafka-style log test"