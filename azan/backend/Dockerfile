FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/azan-api ./cmd/api

FROM alpine:3.20
COPY --from=build /bin/azan-api /bin/azan-api
EXPOSE 8081
ENTRYPOINT ["/bin/azan-api"]
