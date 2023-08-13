FROM golang:1.18-alpine AS backend

RUN apk --no-cache add ca-certificates
# Move to a working directory (/build).
WORKDIR /build

# Copy and download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy a source code to the container.
COPY . .

# Set necessary environmet variables needed for the image and build the server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Run go build (with ldflags to reduce binary size).
RUN go build -ldflags="-s -w" -o bot cmd/main.go

#
# Third stage:
# Creating and running a new scratch container with the backend binary.
#

FROM scratch

# copy the ca-certificate.crt from the build stage
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy binary from /build to the root folder of the scratch container.
COPY --from=backend ["/build/bot", "/"]

# Command to run when starting the container.
ENTRYPOINT ["/bot"]
