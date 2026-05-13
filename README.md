# image-metadata-collector

# Contributing
We are looking forward to contributions. Take a look at our [Contribution Guidelines](CONTRIBUTING.md) before submitting Pull Requests.

# Responsible Disclosure and Security
The [SECURITY.md](SECURITY.md) includes information on responsible disclosure and security related topics like security patches.

# Development
## Local run
```
go run cmd/collector/main.go  --storage fs --environment-name test
```

### Example: image-specific notification overrides
`--image-notification-rules` accepts an ordered JSON array.
The first matching regex wins and replaces the notifications for that image.
Prefix a regex with `!` to match all images that do not match the regex.
The effective priority is:
1. image notification rule
2. notification values from job, pod, or namespace labels/annotations
3. configured default notifications

Example:
```bash
go run cmd/collector/main.go \
  --storage fs \
  --environment-name test \
  --notifications '{"slack":["#security-default"],"emails":["security-default@example.com"],"ms_teams":["default-security-team"]}' \
  --image-notification-rules '[
    {
      "image": "^ghcr\\.io/acme/payment-service:.*$",
      "notifications": {
        "slack": ["#payments-alerts"],
        "emails": ["payments-oncall@example.com"],
        "ms_teams": ["payments-security-team"]
      }
    },
    {
      "image": "^quay\\.io/acme/platform-.+$",
      "notifications": {
        "slack": ["#platform-alerts", "#platform-security"],
        "emails": ["platform-oncall@example.com", "platform-security@example.com"],
        "ms_teams": ["platform-ops-team"]
      }
    },
    {
      "image": "!^quay\\.io/sdase/.*$",
      "notifications": {
        "slack": ["#alerts-cis-5xx"],
        "emails": ["devops+non-quay-images@sda-se.com"],
        "ms_teams": []
      }
    },
    {
      "image": ".*/redis:7(\\..*)?$",
      "notifications": {
        "slack": ["#shared-middleware-alerts"],
        "emails": ["middleware-oncall@example.com"],
        "ms_teams": ["middleware-team"]
      }
    }
  ]'
```

If none of the image regex rules match, the collector uses notification values from job, pod, or namespace labels/annotations.
If no metadata notifications are configured, the collector falls back to `--notifications`.
The collector uses Go's RE2 regex engine, so `!regex` is the supported way to express "all except" matching.

## API upload behavior
When `--storage api` is used, the collector uploads the generated image report to the configured `--api-endpoint`.

The collector currently supports the API endpoint shape:
`https://<host>/v1/account/<accountid>/cluster/<clusterid>/image-collector-report/images`

The multipart endpoints are derived from that URL:
- `POST .../images/upload/init`
- `POST .../images/upload/part`
- direct `PUT` to the presigned S3 URL returned by `upload/part`
- `POST .../images/upload/complete`
- `DELETE .../images/upload` when an initialized multipart upload must be aborted

Upload behavior depends on the final payload size:

1. Small payloads
If the JSON report is 6 MiB or smaller, the collector sends a single `PUT` request to `.../images` with `Content-Type: application/json`.

2. Larger payloads that fit after gzip compression
If the JSON report is larger than 6 MiB, the collector first gzip-compresses it.
If the compressed payload is then 6 MiB or smaller, the collector still uses a single `PUT` request to `.../images` and adds `Content-Encoding: gzip`.

3. Larger payloads that are still above 6 MiB after gzip compression
If the compressed payload is still larger than 6 MiB, the collector switches to multipart upload:
- initialize the multipart upload via `upload/init`
- request one presigned URL per part via `upload/part`
- upload each part directly to S3
- complete the multipart upload via `upload/complete`

4. `413 Request Entity Too Large` from the direct API upload
If the single `PUT` request to `.../images` returns `413 Request Entity Too Large` for a large payload that required compression, the collector retries with the multipart flow.
If a small payload receives `413 Request Entity Too Large`, the collector does not retry with multipart and returns the error.

During multipart completion, the collector reports the final content encoding as:
- `identity` for uncompressed content
- `gzip` for compressed content

Authentication headers such as `x-api-key`, `x-api-signature`, and additional `--http-header` values are sent to the API endpoints.
They are not added to the direct S3 part uploads because those requests use the presigned URL returned by the API.

## Payload contract
The uploaded file keeps its existing top-level structure: a JSON array of image metadata objects.

Each image object now includes `schema_version` so consumers can bind to an explicit payload contract without wrapping the array in another object.
The initial version is `v1`.

The machine-readable contract files are:
- `schema/image-metadata-collector-report-v1.schema.json`
- `schema/image-metadata-collector-report.openapi.yaml`

These files describe the same payload for filesystem output, S3 uploads, and API uploads.

## Test
```
go test ./...
```

## Image Collector Integration Test
To perform integration tests for the image collector, you need a kind cluster:
```bash
cd test_actions/imagecollector
kind delete cluster; kind create cluster && ./setup.bash
```

# Legal Notice
The purpose of the ClusterImageScanner is not to replace the penetration testers or make them obsolete. We strongly recommend running extensive tests by experienced penetration testers on all your applications.
The ClusterImageScanner is to be used only for testing purpose of your running applications/containers. You need a written agreement of the organization of the _environment under scan_ to scan components with the ClusterScanner.

# Author Information
This project is developed by [Signal Iduna](https://www.signal-iduna.de) and [SDA SE](https://sda.se/).
