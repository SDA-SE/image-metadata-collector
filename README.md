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

The collector keeps the existing Kubernetes flags:
- `--kube-config`
- `--kube-context`
- `--master-url`

Supported Kubernetes access modes are:
- in-cluster service account authentication
- kubeconfig with `token` or `tokenFile`
- kubeconfig with `client-certificate` and `client-key`
- CA data from `certificate-authority` or `certificate-authority-data`
- `insecure-skip-tls-verify` when it is present in kubeconfig

Unsupported kubeconfig authentication modes return explicit errors:
- `exec` plugins
- `auth-provider` plugins
- username/password basic auth

Examples:
```bash
go run cmd/collector/main.go \
  --storage fs \
  --environment-name test \
  --kube-config "$HOME/.kube/config" \
  --kube-context my-context
```

```bash
go run cmd/collector/main.go \
  --storage fs \
  --environment-name test \
  --kube-config /path/to/kubeconfig \
  --master-url https://example-cluster-api.internal
```

### Example: image-specific notification overrides
`--image-notification-rules` accepts an ordered JSON array.
The first matching regex wins.
If that rule contains a non-empty `notifications` object, it replaces the notifications for that image.
If that rule contains an empty `notifications` object, the existing notifications stay unchanged and no later rules are applied.
Prefix a regex with `!` to match all images that do not match the regex.
This makes the rules behave like an ordered exception list:
1. put the most specific allow/keep rules first
2. put broader override rules later
3. rely on metadata or `--notifications` only when no rule matches, or when a matching rule has `notifications: {}`

Effective notification priority:
1. first matching image notification rule with a non-empty `notifications` object
2. notification values from job, pod, or namespace labels/annotations
3. configured default notifications from `--notifications`

Example:
```bash
go run cmd/collector/main.go \
  --storage fs \
  --environment-name test \
  --notifications '{"slack":["#security-default"],"emails":["security-default@example.com"],"ms_teams":["default-security-team"]}' \
  --image-notification-rules '[
    {
      "image": "^quay\\.io/sdase/images.*$",
      "notifications": {}
    },
    {
      "image": "^quay\\.io/sdase/.*$",
      "notifications": {
        "slack": ["#alerts-cis-5xx"],
        "emails": ["devops+argocd-images@sda-se.com"],
        "ms_teams": []
      }
    },
    {
      "image": "!^quay\\.io/sdase/.*$",
      "notifications": {
        "slack": ["#alerts-third-party-images"],
        "emails": ["devops+third-party-images@sda-se.com"],
        "ms_teams": []
      }
    }
  ]'
```

With the example above:
- `quay.io/sdase/images-huhu:1.0.0` matches the first rule, keeps namespace or default notifications unchanged, and stops rule evaluation.
- `quay.io/sdase/other-app:1.0.0` does not match the first rule, matches the second rule, and gets `#alerts-cis-5xx` plus `devops+argocd-images@sda-se.com`.
- `docker.io/library/nginx:1.27` matches the third rule and gets the third-party notification targets.

If none of the image regex rules match, the collector uses notification values from job, pod, or namespace labels/annotations.
If a matching image regex rule has an empty `notifications` object, the collector keeps the current notifications from metadata or defaults.
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
