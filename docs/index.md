# Image Metadata Collector

The Image Metadata Collector is a scheduled Kubernetes workload inventory collector. It builds a versioned report of the container images used in a cluster so downstream security-scanning services can select, configure, and notify the right owners about scans.

The report is consumed by the configured storage destination: a filesystem, S3-compatible bucket, Git repository, standard output, or the Image Metadata Collector upload API. The API contract is available in the [OpenAPI definition](https://github.com/SDA-SE/image-metadata-collector/blob/main/schema/image-metadata-collector-report.openapi.yaml).

## Kubernetes discovery

Each run lists namespaces and then discovers images from Pods, Jobs, and CronJobs in every namespace. Pod container and init-container images include the runtime image ID when Kubernetes reports it. Job and CronJob template images are included even when no workload instance is running.

For every discovered workload, the collector merges workload labels and annotations with namespace metadata. That metadata supplies report fields such as scan settings, product, team, owners, and notification destinations. Command-line defaults fill in values that are not supplied by metadata.

## Deployment and access

The repository provides a Kubernetes CronJob deployment in [`deployment/base`](https://github.com/SDA-SE/image-metadata-collector/tree/main/deployment/base). It uses in-cluster service-account authentication by default; local execution can instead use `--kube-config`, `--kube-context`, and `--master-url`.

Full discovery requires cluster-scoped, read-only `get` and `list` permissions for:

- core resources: `namespaces` and `pods`
- `batch/v1` resources: `jobs` and `cronjobs`

The current bundled ClusterRole grants only `pods` and `namespaces`. Extend it with the required `batch/v1` permissions before relying on Job and CronJob discovery in a deployed collector.

The CronJob is configured with `concurrencyPolicy: Forbid`, which prevents overlapping scheduled runs. Configure the API endpoint and signature through the ConfigMap, and the API key through the Secret, as shown in the deployment manifests.

## Configuration and storage

Configuration is exposed as command-line flags and can also be supplied through environment variables using the `IMAGE_METADATA_COLLECTOR_` prefix; dashes in flag names become underscores. Important configuration paths are Kubernetes connection flags (`--kube-config`, `--kube-context`, `--master-url`), collector defaults such as `--environment-name`, and metadata annotation-name prefixes.

Select the report destination with `--storage`. Supported values are `api`, `s3`, `git`, `fs`, and `stdout`. `--filename` applies to filesystem, S3, and Git storage; API uploads derive their resource name from `--project` when set.

Notification values are resolved in this order:

1. The first matching `--image-notification-rules` entry with a non-empty `notifications` object replaces the value.
2. Otherwise, notification values from Job, Pod, or Namespace metadata are used.
3. If metadata does not provide a value, `--notifications` supplies the default.

Rules are evaluated in order and the first match always ends evaluation. A matching rule with an empty `notifications` object preserves the current metadata or default value. Prefixing a regular expression with `!` matches images that do not match that expression.

## Report and upload contract

Reports are JSON arrays of image metadata objects. Every object has a `schema_version`; the current payload version is `v1`. This contract is shared by filesystem, S3, Git, and API output, and is defined by the [OpenAPI definition](https://github.com/SDA-SE/image-metadata-collector/blob/main/schema/image-metadata-collector-report.openapi.yaml) and [JSON Schema](https://github.com/SDA-SE/image-metadata-collector/blob/main/schema/image-metadata-collector-report-v1.schema.json).

For API storage, configure `--api-endpoint` with the base `.../image-collector-report/images` endpoint. The collector uploads JSON directly when the final payload is at most 6 MiB. Larger reports are gzip-compressed first; if the compressed report still exceeds 6 MiB, the collector uses the multipart upload endpoints. A `413 Request Entity Too Large` response to a compressed direct upload also causes a multipart retry. With `--project`, the target changes from `images` to `images_<project>` for both direct and multipart uploads.

## Operations

Run the collector on a schedule appropriate for workload churn and monitor its logs for Kubernetes discovery or storage-upload failures. Keep API credentials in Kubernetes Secrets, grant only the read permissions listed above, and validate report consumers against the versioned contract before changing payload handling.

For development and detailed flag examples, see the repository [README](https://github.com/SDA-SE/image-metadata-collector/blob/main/README.md). For the machine-readable upload contract, see the [OpenAPI definition](https://github.com/SDA-SE/image-metadata-collector/blob/main/schema/image-metadata-collector-report.openapi.yaml).
