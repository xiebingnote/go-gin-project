# Makefile for the kernel module
# Home directory variable
HOME_DIR := $(shell pwd)

# Output directory variable
OUT_DIR := $(HOME_DIR)/output

# Application name variable
APP_NAME := go_gin_project

# Image name variable
IMAGE_NAME:= go_gin_project

# Container name variable
CONTAINER_NAME:= go_gin_project

# Get the container id
CONTAINER_ID := $(shell docker ps -a -q -f name=$(CONTAINER_NAME))

# Get the image id
IMAGE_ID := $(shell docker images -q $(IMAGE_NAME))

# GOPKGS variable
GOPKGS := $(shell go list ./... | grep -v /vendor/)

# GOPATH environment variable
export GOENV= $(HOME_DIR)/go.env

# Makefile targets (file to create)
all: prepare compile test package image run

# Clean the output directory
clean: clean_dir clean_image clean_docker

# Makefile phony targets (no file to create)
.phony: prepare compile test package image clean_dir clean_docker clean_image

# Prepare the environment, Check the environment
prepare:
	# Check git version
	git version
	# Check go environment
	go env
	# Check go modules and download if necessary
	go mod tidy || go mod download || go mod download -X

# Compile the application
compile:build

# Build the application binary file
build:
	# Compile the application
	go build -o $(HOME_DIR)/bin/$(APP_NAME)

# Run the tests and generate the coverage report
test:
	# Run the tests
	go test --race -timeout=300s -v -cover $(GOPKGS) -coverprofile=coverage.out | tee unittest.text

# Package the application for distribution
package:
	$(shell rm -rf $(OUT_DIR))
	$(shell mkdir -p $(OUT_DIR))
	$(shell mkdir -p $(OUT_DIR)/var/)
	$(shell cp -a bin $(OUT_DIR)/bin)
	$(shell cp -a conf $(OUT_DIR)/conf)
	$(shell if [ -d "script" ]; then cp -r script $(OUT_DIR)/script; fi)
	$(shell if [ -d "webroot" ]; then cp -r webroot $(OUT_DIR)/webroot; fi)
	tree $(OUT_DIR)

# Build the docker image
image:
	docker build --build-arg DIRNAME=$(shell basename $(PWD)) -t $(IMAGE_NAME) .

# Run the application in a docker container
run:
	docker run --name $(CONTAINER_NAME) -p 8080:8080 -d $(IMAGE_NAME)

# Clean the directory
clean_dir:
	$(shell if [ -f "coverage.out" ]; then rm -f coverage.out; fi)
	$(shell if [ -f "unittest.text" ]; then rm -f unittest.text; fi)
	@rm -rf $(OUT_DIR)

# Clean the docker container
# Clean the docker container and image
clean_docker:
ifneq ($(CONTAINER_ID),)
	@echo "Delete Container $(CONTAINER_NAME)..."
	docker rm -f $(CONTAINER_ID)
endif

# Clean the docker image
clean_image:
ifneq ($(IMAGE_ID),)
	@echo "Delete Image $(IMAGE_NAME)..."
	docker rmi -f $(IMAGE_ID)
endif