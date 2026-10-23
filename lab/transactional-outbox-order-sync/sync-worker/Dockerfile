FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
# Static binary so it runs on an empty base image.
RUN CGO_ENABLED=0 go build -o /sync-worker .

FROM scratch
COPY --from=build /sync-worker /sync-worker
ENTRYPOINT ["/sync-worker"]
