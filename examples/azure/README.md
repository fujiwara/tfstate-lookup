# Azure Blob Storage Example

This example demonstrates how to use tfstate-lookup with Azure Blob Storage.

## Prerequisites

- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) installed
- An Azure Storage Account with a terraform.tfstate file

## Authentication

tfstate-lookup uses [DefaultAzureCredential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential) for Azure authentication.

### Option 1: Azure CLI (for local development)

```bash
az login --scope https://management.core.windows.net//.default
```

> **Note**: If MFA (Multi-Factor Authentication) is enabled, you must use the `--scope` option.

### Option 2: Environment variables (for CI/CD or service principal)

```bash
export AZURE_TENANT_ID=your-tenant-id
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-client-secret
```

### Option 3: Managed Identity (for Azure VMs or containers)

No configuration needed. DefaultAzureCredential will automatically use managed identity.

## Usage

### With Azure AD authentication

```bash
export ARM_USE_AZUREAD=true
tfstate-lookup -s azurerm://resource-group/storage-account/container/terraform.tfstate
```

### With subscription ID (if needed)

```bash
export AZURE_SUBSCRIPTION_ID=your-subscription-id
tfstate-lookup -s azurerm://resource-group/storage-account/container/terraform.tfstate
```

### URL format with subscription ID

```bash
tfstate-lookup -s azurerm://subscription-id@resource-group/storage-account/container/terraform.tfstate
```

## Troubleshooting

### Error: "you must use multi-factor authentication"

Re-authenticate with the management scope:
```bash
az login --scope https://management.core.windows.net//.default
```

### Error: "failed to list keys" or permission denied

Ensure the authenticated account has one of the following:
- `Storage Blob Data Reader` role on the storage account (for Azure AD auth)
- `Reader` and `Storage Account Key Operator Service Role` on the storage account (for access key auth)

### Error: "missing environment variable AZURE_TENANT_ID"

Either:
1. Log in via Azure CLI: `az login`
2. Set the required environment variables for service principal authentication
