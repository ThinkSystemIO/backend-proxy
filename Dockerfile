FROM golang:alpine as build

# Download and use git
ARG PAT
RUN apk add git
RUN git config --global url.https://${PAT}:@github.com/.insteadOf https://github.com/

# Set necessary env variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPRIVATE=github.com/ThinkSystemIO

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o main .

# Create final container to hide history
FROM golang:alpine

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
COPY --from=build /build/main .

# Export necessary port
EXPOSE 80

# Command to run when starting the container
CMD ["/dist/main"]