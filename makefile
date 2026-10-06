WEB_DIR = ./web
API_DIR = .
DEV_WEB_PORT ?= 5173
DEV_SQLITE_PATH ?= one-api.db

.PHONY: all build-web build-all-web start-api dev dev-api dev-web reset-setup test

all: build-all-web start-api

build-web:
	@echo "Building web frontend..."
	@cd $(WEB_DIR) && bun install --frozen-lockfile
	@cd $(WEB_DIR) && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$$(cat ../VERSION) bun run build

build-all-web: build-web

start-api:
	@echo "Starting api dev server..."
	@cd $(API_DIR) && go run main.go &

# 本机开发直跑（本项目唯一部署方式是 docker run + 公开镜像，仓库内不再有 compose/dev 栈）
dev-api:
	@echo "Starting api dev server locally (go run); default DB is SQLite: $(DEV_SQLITE_PATH)"
	@echo "  override with SQLITE_PATH=$(DEV_SQLITE_PATH) or SQL_DSN=<mysql/postgres dsn>"
	@cd $(API_DIR) && go run main.go &

dev-web:
	@echo "Starting web frontend dev server..."
	@echo "Web frontend: http://localhost:$(DEV_WEB_PORT)"
	@cd $(WEB_DIR) && bun install
	@cd $(WEB_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_WEB_PORT)

dev: dev-api dev-web

# The main package embeds the ignored web/dist output and is covered after build-web.
test:
	@echo "Testing root Go module..."
	@root_module=$$(GOWORK=off go list -m); \
		root_packages=$$(GOWORK=off go list -e ./... | grep -vxF "$$root_module"); \
		GOWORK=off go test $$root_packages
	@echo "Testing relaykit Go module..."
	@cd relaykit && GOWORK=off go test ./...

reset-setup:
	@echo "Resetting local setup wizard state (SQLite)..."
	@db_path="$${SQLITE_PATH:-$(DEV_SQLITE_PATH)}"; db_path="$${db_path%%\?*}"; \
	if [ -f "$$db_path" ]; then \
		echo "Detected local SQLite database: $$db_path"; \
		sqlite3 "$$db_path" \
			"DELETE FROM setups; DELETE FROM users WHERE role = 100; DELETE FROM options WHERE key IN ('SelfUseModeEnabled', 'DemoSiteEnabled');"; \
		echo "SQLite setup state reset. Restart the local api process before testing the setup wizard."; \
	else \
		echo "No local SQLite database found at $$db_path."; \
		echo "Start the api with 'make dev-api', or set SQLITE_PATH/DEV_SQLITE_PATH to your SQLite database."; \
		exit 1; \
	fi
