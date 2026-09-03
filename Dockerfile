FROM docker.arvancloud.ir/golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM docker.arvancloud.ir/alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /api
EXPOSE 8090
ENTRYPOINT ["/api"]
