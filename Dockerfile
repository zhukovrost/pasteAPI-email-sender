# Working on dependencies
FROM golang:1.22-alpine3.18 as modules

# Copy dependencies list to modules directory
COPY go.mod go.sum /modules/

# Set the working directory inside the container
WORKDIR /modules

# Download dependencies
RUN go mod download

# Start with the official Go image as a base image
FROM golang:1.22-alpine as builder

# Copy cached modules
COPY --from=modules /go/pkg /go/pkg

# Copy the rest of the application code to the working directory
COPY . /app

# Set the working directory inside the container
WORKDIR /app

# Ensure that bash and make are installed
RUN apk add --no-cache bash make

# Build the application using the Makefile
RUN make build/app

# Create a new stage for a minimal runtime image
FROM scratch

# Copy configuration files, if necessary
COPY --from=builder /app/configs /configs

# Copy binary file
COPY --from=builder /app/bin/app /app

# Run the application
CMD ["/app"]
