FROM golang:1.27.2-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server-go .

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/server-go /app/server-go
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server-go"]