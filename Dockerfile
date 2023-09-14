FROM golang:1.21 as build-env
WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/collector/main.go && \
  go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.4.1 && \
  cyclonedx-gomod mod -json=true -output /bom.json

FROM gcr.io/distroless/static-debian11
COPY --from=build-env /go/bin/app /
COPY --from=build-env /bom.json /bom.json

USER 1001

ENV ANNOTATION_NAME_ENGAGEMENT_TAG="clusterscanner.sdase.org/engagement-tags"
ENV DEFAULT_ENGAGEMENT_TAGS="cluster-image-scanner"
ENV ANNOTATION_NAME_PRODUCT="contact.sdase.org/product"
ENV ANNOTATION_NAME_SLACK="contact.sdase.org/slack"
ENV ANNOTATION_NAME_EMAIL="contact.sdase.org/email"
ENV ANNOTATION_NAME_TEAM="contact.sdase.org/team"
ENV ANNOTATION_NAME_ROCKETCHAT="contact.sdase.org/rocketchat"
ENV ANNOTATION_NAME_CONTAINER_TYPE="contact.sdase.org/container_type"
ENV ANNOTATION_NAME_NAMESPACE_FILTER="clusterscanner.sdase.org/namespace_filter"
ENV ANNOTATION_NAME_NAMESPACE_FILTER_NEGATED="clusterscanner.sdase.org/negated_namespace_filter"
ENV DEFAULT_TEAM_NAME="nobody"
ENTRYPOINT ["/app"]
