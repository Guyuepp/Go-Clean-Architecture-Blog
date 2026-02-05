# Database
MYSQL_USER ?= user
MYSQL_PASSWORD ?= password
MYSQL_ADDRESS ?= 127.0.0.1:3306
MYSQL_DATABASE ?= article

# Exporting bin folder to the path for makefile
export PATH   := $(PWD)/bin:$(PATH)
# Default Shell
export SHELL  := bash
# Type of OS: Linux or Darwin.
export OSTYPE := $(shell uname -s | tr A-Z a-z)
export ARCH := $(shell uname -m)



# --- Tooling & Variables ----------------------------------------------------------------
include ./misc/make/tools.Makefile
include ./misc/make/help.Makefile

# ~~~ Development Environment ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

up: dev-env dev-air             ## Startup / Spinup Docker Compose and air
down: docker-stop               ## Stop Docker
destroy: docker-teardown clean  ## Teardown (removes volumes, tmp files, etc...)

install-deps: migrate air golangci-lint ## Install Development Dependencies (locally).
deps: $(MIGRATE) $(AIR) $(GOLANGCI) ## Checks for Global Development Dependencies.
deps:
	@echo "Required Tools Are Available"

dev-env: ## Bootstrap Environment (with Docker Compose help).
	@ docker compose up -d --build mysql
	@ docker compose up -d --build redis
	@echo "Waiting for services to be healthy..."
	@sleep 5

dev-env-test: dev-env ## Run application (within Docker Compose help)
	@ $(MAKE) image-build
	@ docker compose up web

dev-air: $(AIR) ## Starts AIR (Continuous Development app).
	@ air

dev-run: ## Run the application directly
	@ go run ./app/main.go

docker-stop:
	@ docker compose down

docker-teardown:
	@ docker compose down --remove-orphans -v

# ~~~ Code Actions ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

lint: $(GOLANGCI) ## Runs golangci-lint with predefined configuration
	@echo "Applying linter"
	@ golangci-lint version
	@ golangci-lint run -c .golangci.yaml ./...

lint-fix: $(GOLANGCI) ## Runs golangci-lint and fixes issues automatically
	@echo "Applying linter with auto-fix"
	@ golangci-lint run -c .golangci.yaml --fix ./...

# -trimpath - will remove the filepaths from the reports, good to save money on network traffic,
#             focus on bug reports, and find issues fast.
# - race    - adds a racedetector, in case of racecondition, you can catch report with sentry.
#             https://golang.org/doc/articles/race_detector.html
#
# todo(butuzov): add additional flags to compiler to have a `version` flag.
build: ## Builds binary
	@ printf "Building application... "
	@ go build \
		-trimpath  \
		-o engine \
		./app/
	@ echo "done"


build-race: ## Builds binary (with -race flag)
	@ printf "Building application with race flag... "
	@ go build \
		-trimpath  \
		-race      \
		-o engine \
		./app/
	@ echo "done"

# ~~~ Docker Build ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.ONESHELL:
image-build: ## Build Docker image
	@ echo "Docker Build"
	@ docker build \
		--file Dockerfile \
		--tag go-clean-arch \
			.

image-run: image-build ## Build and run Docker image
	@ docker compose up -d mysql redis
	@echo "Waiting for database..."
	@sleep 5
	@ docker compose up web

# Commenting this as this not relevant for the project, we load the DB data from the SQL file.
# please refer this when introducing the database schema migrations.

# # ~~~ Database Migrations ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 

# MYSQL_DSN := "mysql://$(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_ADDRESS))/$(MYSQL_DATABASE)"

# migrate-up: $(MIGRATE) ## Apply all (or N up) migrations.
# 	@ read -p "How many migration you wants to perform (default value: [all]): " N; \
# 	migrate  -database $(MYSQL_DSN) -path=misc/migrations up ${NN}

# .PHONY: migrate-down
# migrate-down: $(MIGRATE) ## Apply all (or N down) migrations.
# 	@ read -p "How many migration you wants to perform (default value: [all]): " N; \
# 	migrate  -database $(MYSQL_DSN) -path=misc/migrations down ${NN}

# .PHONY: migrate-drop
# migrate-drop: $(MIGRATE) ## Drop everything inside the database.
# 	migrate  -database $(MYSQL_DSN) -path=misc/migrations drop

# .PHONY: migrate-create
# migrate-create: $(MIGRATE) ## Create a set of up/down migrations with a specified name.
# 	@ read -p "Please provide name for the migration: " Name; \
# 	migrate create -ext sql -dir misc/migrations $${Name}

# ~~~ Cleans ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

clean: clean-artifacts clean-docker ## Clean all artifacts and docker resources

clean-artifacts: ## Removes Artifacts (*.out, engine binary)
	@ printf "Cleaning artifacts... "
	@ rm -f *.out engine
	@ echo "done."

clean-docker: ## Removes dangling docker images
	@ docker image prune -f

clean-all: clean ## Deep clean including dependencies cache
	@ go clean -cache -testcache -modcache
	@ echo "All caches cleared."

fmt: ## Format code with gofmt
	@ echo "Formatting code..."
	@ gofmt -s -w .
	@ echo "done."

vet: ## Run go vet
	@ echo "Running go vet..."
	@ go vet ./...
	@ echo "done."

mod-tidy: ## Tidy go modules
	@ echo "Tidying go modules..."
	@ go mod tidy
	@ echo "done."

mod-download: ## Download go modules
	@ echo "Downloading go modules..."
	@ go mod download
	@ echo "done."
