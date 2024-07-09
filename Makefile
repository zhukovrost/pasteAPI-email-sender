# Include variables from the .envrc file
include .env

# ==================================================================================== #
# RUN
# ==================================================================================== #

## run/app: run the application
.PHONY: run/app
run/app: build/app
	@echo 'Running app...'
	./bin/app

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: # vendor
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/app: build the cmd/app application
.PHONY: build/app
build/app: # build/docs
	@echo 'Building cmd/api...'
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64
	go build -o=./bin/app ./cmd/app

## build/image: build Docker image for the application
.PHONY: build/image
build/image:
	@echo 'Building image...'
	-docker compose down
	-docker rmi zhukovrost/pasteapi-email-sender:${VERSION} 2>/dev/null || true
	docker build -t zhukovrost/pasteapi-email-sender .
