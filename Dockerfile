# Dockerfile for go-gin-project
# This Dockerfile is used to build a Docker image for go-gin-project.
FROM centos:latest

# Labels for the image to provide additional information
LABEL maintainer="xiebing <xiebing@xxx.com>"
LABEL description="Go Gin Project Docker Image"
LABEL version="1.0"

# Define the environment variable for the application name
ENV APPNAME=go_gin_project

# Define the environment variable for the application directory name
ARG DIRNAME
ENV DIRNAME=${DIRNAME}

# Update the system and install the required packages
USER root

# Set the permissions for the application binary
RUN useradd -ms /bin/bash work \
    && mkdir -p /home/work/${DIRNAME} \
    && chown -R work:work /home/work/${DIRNAME}/

# Copy the application binary and configuration files to the image
COPY output/ /home/work/$DIRNAME/

# Set the permissions for the application binary
RUN chmod -R 755 /home/work/${DIRNAME}/bin/${APPNAME}

# Set the working directory for the application to be built and run
USER work

WORKDIR /home/work/
# Set the entrypoint command to run the application
CMD ["sh", "-c", "/home/work/${DIRNAME}/bin/${APPNAME}"]