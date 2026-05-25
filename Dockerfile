FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /app/server /app/server
COPY templates ./templates
EXPOSE 8080
ENTRYPOINT ["/app/server"]
