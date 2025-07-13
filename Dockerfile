# Use an official Go runtime as a parent image
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum .
RUN go mod download

# Copy the Go application source code into the container
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 go build -o git-clone .

# Start a new stage for the minimal runtime container
FROM gcr.io/distroless/static-debian12

# Set the working directory inside the minimal runtime container
WORKDIR /app

# Copy the built binary from the builder container into the minimal runtime container
COPY --from=builder /app/git-clone . 

# Run your Go application and then sleep indefinitely
ENTRYPOINT ["/app/git-clone"]
