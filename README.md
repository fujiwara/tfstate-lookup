# tfstate-lookup

Lookup resource attributes in tfstate.

## Install

### homebrew

```
$ brew install fujiwara/tap/tfstate-lookup
```

### [Binary releases](https://github.com/fujiwara/tfstate-lookup/releases)

## Usage (command)

```
Usage of tfstate-lookup:
  -dump
        dump all resources
  -i    interactive mode
  -j    run jid after selecting an item
  -s string
        tfstate file path or URL (default "terraform.tfstate")
  -s3-endpoint-url string
        S3 endpoint URL
  -state string
        tfstate file path or URL (default "terraform.tfstate")
  -timeout duration
        timeout for reading tfstate
```

Supported URL schemes are http(s), s3, gs, azurerm, file or remote (for Terraform Cloud and Terraform Enterprise).

```console
$ tfstate-lookup -s .terraform/terraform.tfstate aws_vpc.main.id
vpc-1a2b3c4d

$ tfstate-lookup aws_vpc.main
{
  "arn": "arn:aws:ec2:ap-northeast-1:123456789012:vpc/vpc-1a2b3c4d",
  "assign_generated_ipv6_cidr_block": false,
  "cidr_block": "10.0.0.0/16",
  "default_network_acl_id": "acl-001234567890abcde",
  "default_route_table_id": "rtb-001234567890abcde",
  "default_security_group_id": "sg-01234567890abcdef",
  "dhcp_options_id": "dopt-64569903",
  "enable_classiclink": false,
  "enable_classiclink_dns_support": false,
  "enable_dns_hostnames": true,
  "enable_dns_support": true,
  "id": "vpc-1a2b3c4d",
  "instance_tenancy": "default",
  "ipv6_association_id": "",
  "ipv6_cidr_block": "",
  "main_route_table_id": "rtb-001234567890abcde",
  "owner_id": "123456789012",
  "tags": {
    "Name": "main"
  }
}
```

A remote state is supported only S3, GCS, AzureRM and Terraform Cloud / Terraform Enterprise backend currently.

### Parent key access for indexed resources

You can access parent keys of resources defined with `count` or `for_each` to get all instances at once.

#### Count resources (array)

```console
$ tfstate-lookup aws_instance.web
[
  {
    "id": "i-1234567890abcdef0",
    "instance_type": "t3.micro",
    ...
  },
  {
    "id": "i-0987654321fedcba0", 
    "instance_type": "t3.micro",
    ...
  }
]

$ tfstate-lookup aws_instance.web[0].id
i-1234567890abcdef0
```

#### For_each resources (map)

```console
$ tfstate-lookup aws_s3_bucket.example
{
  "staging": {
    "id": "my-bucket-staging",
    "bucket": "my-bucket-staging",
    ...
  },
  "production": {
    "id": "my-bucket-production",
    "bucket": "my-bucket-production", 
    ...
  }
}

$ tfstate-lookup 'aws_s3_bucket.example["staging"].id'
my-bucket-staging
```

### Interactive mode

You can use interactive mode with `-i` option.

```console
$ tfstate-lookup -i
Search: █
? Select an item: 
  ▸ aws_acm_certificate.foo
    aws_acm_certificate_validation.foo
    aws_cloudwatch_log_group.foo
    aws_ecs_cluster.foo
...
```

When you select an item, it shows the attributes of the resource.

### Run jid after selecting an item

You can run [jid](https://github.com/simeji/jid) after selecting an item with `-j` option.

jid is a JSON incremental digger.

`tfstate-lookup -i -j` runs jid after selecting an item.

`tfstate-lookup -j some.resource` runs jid for the attributes of the resource.

tfstate-lookup integrates jid as a library, so you don't need to install jid command.

See also [simiji/jid](https://github.com/simeji/jid).

### Dump all resources, outputs, and data sources in tfstate

You can dump all resources, outputs, and data sources in tfstate with `-dump` option.

```console
$ tfstate-lookup -dump
```

The output is a JSON object. The keys are the address of the resource, output, or data source. The values are the same as the lookup result.

```jsonnet
{
  "aws_vpc.main": {
    "arn": "arn:aws:ec2:ap-northeast-1:123456789012:vpc/vpc-1a2b3c4d",
    "assign_generated_ipv6_cidr_block": false,
    // ...
  },
  "data.aws_ami.foo": {
    "arn": "arn:aws:ec2:ap-northeast-1:123456789012:ami/ami-1a2b3c4d",
    // ...
  },
  "output.foo": "bar",
  // ...
}
```

## Usage (Go package)

See details in [godoc](https://pkg.go.dev/github.com/fujiwara/tfstate-lookup/tfstate).

```go
package main

import(
    "fmt"
    "os"

    "github.com/fujiwara/tfstate-lookup/tfstate"
)

func main() {
    state, _ := tfstate.ReadURL(ctx, "s3://mybucket/terraform.tfstate")
    attrs, _ := state.Lookup("aws_vpc.main.id")
    fmt.Println(attrs.String())
}
```


## Supported tfstate URL format

- Local file `file://path/to/terraform.tfstate`
- HTTP/HTTPS `https://example.com/terraform.tfstate`
- Amazon S3 `s3://{bucket}/{key}`
- Terraform Cloud `remote://app.terraform.io/{organization}/{workspaces}`
  - `TFE_TOKEN` environment variable is required.
- Google Cloud Storage `gs://{bucket}/{key}`
- Azure Blog Storage
  - `azurerm://{resource_group_name}/{storage_account_name}/{container_name}/{blob_name}`
  - `azurerm://{subscription_id}@{resource_group_name}/{storage_account_name}/{container_name}/{blob_name}`

### S3 endpoint URL support

You can specify the S3 endpoint URL with `-s3-endpoint-url` option. `AWS_ENDPOINT_URL_S3` environment variable is also supported.

```console
$ tfstate-lookup -s3-endpoint-url http://localhost:9000 s3://mybucket/terraform.tfstate

$ AWS_ENDPOINT_URL_S3=http://localhost:9000 tfstate-lookup s3://mybucket/terraform.tfstate
```

This option is useful for S3 compatible storage services.

### Terraform Workspace support

You can specify the Terraform workspace with `TF_WORKSPACE` environment variable.

## LICENSE

[Mozilla Public License Version 2.0](LICENSE)
