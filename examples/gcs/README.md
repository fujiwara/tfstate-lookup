# Google Cloud Storage Example

This example demonstrates how to use tfstate-lookup with Google Cloud Storage (GCS).

## Prerequisites

- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) installed
- A GCS bucket with a terraform.tfstate file

## Authentication

tfstate-lookup uses [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials) for GCS authentication.

### Option 1: User credentials (for local development)

```bash
gcloud auth application-default login
```

> **Note**: `gcloud auth login` is not sufficient. You must use `gcloud auth application-default login` for Go client libraries.

### Option 2: Service account (for CI/CD or production)

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

## Usage

```bash
# Read tfstate from GCS
tfstate-lookup -s gs://your-bucket/path/to/terraform.tfstate

# Lookup a specific resource
tfstate-lookup -s gs://your-bucket/path/to/terraform.tfstate aws_vpc.main.id
```

## Troubleshooting

### Error: "invalid_grant" "Bad Request"

This error occurs when:
- Application Default Credentials are not set up
- The credentials have expired

Solution:
```bash
gcloud auth application-default login
```

### Error: Permission denied

Ensure the authenticated account has the `storage.objects.get` permission on the bucket.
