FROM golang:alpine

WORKDIR /app

# Download necessary Go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy files and build
COPY *.go .
RUN go build -o /mirth_channel_exporter

# Runtime
EXPOSE 8080
CMD [ "/mirth_channel_exporter" ]
