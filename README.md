# tfstate-lookup

Lookup resource attributes in tfstate.

## Usage (Go package)

See details in [godoc](https://godoc.org/github.com/fujiwara/tfstate-lookup/tfstate).

```go
package main

import(
    "fmt"
    "os"

    "github.com/fujiwara/tfstate-lookup/tfstate"
)

func main() {
	f, _ := os.Open("terraform.tfstate")
    state, _ := tfstate.Read(f)
    res, _ := state.Lookup("aws_vpc.main.id")
    vpcID := res.(string)
    fmt.Println(vpcID)
}
```

## Usage (command)

```
Usage of tfstate-lookup:
  -r	raw output
  -raw
    	raw output
  -s string
    	tfstate file path (default "terraform.tfstate")
  -state string
    	tfstate file path (default "terraform.tfstate")
```

```console
$ tfstate-lookup aws_vpc.main.id
"vpc-1a2b3c4d"

$ tfstate-lookup -r aws_vpc.main.id
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

## LICENSE

[Mozilla Public License Version 2.0](LICENSE)
