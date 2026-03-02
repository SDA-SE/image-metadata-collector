FROM golang:1.26.0 AS build-env
WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/collector/main.go && \
  go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0 && \
  cyclonedx-gomod mod -json=true -output /bom.json

FROM gcr.io/distroless/static-debian12
COPY --from=build-env /go/bin/app /
COPY --from=build-env /bom.json /bom.json

USER 1001

ENV COLLECTOR_ENGAGEMENT_TAGS="cluster-image-scanner"
ENTRYPOINT ["/app"]
