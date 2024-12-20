FROM golang:1.23 as build-env
WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/collector/main.go && \
  go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.4.1 && \
  cyclonedx-gomod mod -json=true -output /bom.json

FROM gcr.io/distroless/static-debian12
COPY --from=build-env /go/bin/app /
COPY --from=build-env /bom.json /bom.json

USER 1001

ENV COLLECTOR_ENGAGEMENT_TAGS="cluster-image-scanner"
ENTRYPOINT ["/app"]
