# Makefile for the kernel module
# Home directory variable
HOMEDIR := $(shell pwd)
# Output directory variable
OUTDIR := $(HOMEDIR)/output
# Application name variable
APPNAME := go_gin_project
# Image name variable
IMAGENAME:= go_gin_project
# Container name variable
CONTAINERNAME:= go_gin_project
# GOPKGS variable
GOPKGS := $(shell go list ./... | grep -v /vendor/)
# GOPATH environment variable
export GOENV= $(HOMEDIR)/go.env
# Makefile targets (file to create)
all: compile package image run
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
	go build -o $(HOMEDIR)/bin/$(APPNAME)
# Run the tests and generate the coverage report
test:
	# Run the tests
	go test --race -timeout=300s -v -cover $(GOPKGS) -coverprofile=coverage.out | tee unittest.text
# Package the application for distribution
package:
	$(shell rm -rf $(OUTDIR))
	$(shell mkdir -p $(OUTDIR))
	$(shell mkdir -p $(OUTDIR)/var/)
	$(shell cp -a bin $(OUTDIR)/bin)
	$(shell cp -a conf $(OUTDIR)/conf)
	$(shell if [ -d "script" ]; then cp -r script $(OUTDIR)/script; fi)
	$(shell if [ -d "webroot" ]; then cp -r webroot $(OUTDIR)/webroot; fi)
	tree $(OUTDIR)
# Build the docker image
image:
	docker build --build-arg DIRNAME=$(shell basename $(PWD)) -t $(IMAGENAME) .
# Run the application in a docker container
run:
	docker run --name $(CONTAINERNAME) -p 8080:8080 -d $(IMAGENAME)
# Clean the directory
clean_dir:
	$(shell if [ -f "coverage.out" ]; then rm -f coverage.out; fi)
	$(shell if [ -f "unittest.text" ]; then rm -f unittest.text; fi)
	@rm -rf $(OUTDIR)
# Clean the docker container
# Get the container id
CONTAINER_ID := $(shell docker ps -a -q -f name=$(CONTAINERNAME))
# Get the image id
IMAGE_ID := $(shell docker images -q $(IMAGENAME))
# Clean the docker container and image
clean_docker:
ifneq ($(CONTAINER_ID),)
	@echo "Delete Container $(CONTAINERNAME)..."
	docker rm -f $(CONTAINER_ID)
endif
# Clean the docker image
clean_image:
ifneq ($(IMAGE_ID),)
	@echo "Delete Image $(IMAGENAME)..."
	docker rmi -f $(IMAGE_ID)
endif