FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.21
COPY --from=build /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
