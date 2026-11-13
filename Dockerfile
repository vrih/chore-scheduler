# go-sqlite3 needs cgo, so build on Alpine (musl) and link statically.
FROM golang:1.24-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath \
      -ldflags "-s -w -linkmode external -extldflags -static" \
      -o /out/chore-scheduler ./cmd/chore-scheduler

FROM alpine:3.20
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /out/chore-scheduler /usr/local/bin/chore-scheduler
ENV CHORE_SCHEDULER_DB=/data/chore.db
EXPOSE 8080
ENTRYPOINT ["chore-scheduler"]
CMD ["serve", "--addr", "0.0.0.0:8080"]
