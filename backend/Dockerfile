FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /transitops ./cmd/server

FROM alpine:3.22

WORKDIR /app
COPY --from=build /transitops /transitops

EXPOSE 8080
CMD ["/transitops"]
