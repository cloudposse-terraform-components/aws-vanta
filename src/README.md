---
tags:
  - component/vanta
  - layer/security-and-compliance
  - provider/aws
---

# Component: `vanta`

This component provisions the `vanta-auditor` IAM role in each AWS account, enabling the
[Vanta](https://www.vanta.com/) GRC (Governance, Risk, and Compliance) platform to continuously scan AWS resource
configurations. Vanta uses this read-only access to map evidence to compliance frameworks (SOC 2, ISO 27001, HIPAA,
PCI DSS, GDPR, etc.).

**Important**: This component only provisions the IAM role and policies on the AWS side. Configuration, administration,
and operation of the Vanta platform itself is managed externally.

## Component Features

This component is responsible for:

- **Cross-Account IAM Role**: Creates a `vanta-auditor` IAM role that Vanta's scanner role can assume
- **Read-Only Access**: Attaches the AWS managed `SecurityAudit` policy for broad read-only security access
- **Supplemental Permissions**: Creates and attaches a `VantaAdditionalPermissions` custom policy with IAM Identity
  Center read access and explicit deny on sensitive data (RDS logs, DataPipeline data)
- **Management Account Support**: Optionally attaches `VantaManagementAccountPermissions` for organization-level reads
  (only in the root account)
- **External ID Protection**: Uses an external ID in the trust policy to prevent confused deputy attacks
- **Account Verification**: Optional safety check that validates Terraform is running in the correct AWS account

## Key Capabilities

- **Read-Only**: Vanta cannot modify any AWS resources — all permissions are read-only or describe/list
- **Metadata Only**: Scans resource configurations and metadata, not application data
- **Automatic Evidence Collection**: Vanta continuously maps AWS configurations to compliance controls
- **Multi-Framework Support**: SOC 2, ISO 27001, HIPAA, PCI DSS, GDPR, and custom frameworks
- **Audit Trail**: All Vanta API calls are logged in CloudTrail as `AssumeRole` from Vanta's scanner roles

## Architecture

The component deploys a simple per-account IAM role with no cross-account dependencies:

```text
                     AWS Organization

   Vanta Platform (scanner roles from multiple Vanta AWS accounts)
        │
        │  sts:AssumeRole (with ExternalId)
        │
        ├──► management account ── vanta-auditor role + SecurityAudit
        │                          + VantaAdditionalPermissions
        │                          + VantaManagementAccountPermissions
        │
        ├──► security account   ── vanta-auditor role + SecurityAudit
        │                          + VantaAdditionalPermissions
        │
        ├──► audit account      ── vanta-auditor role + SecurityAudit
        │                          + VantaAdditionalPermissions
        │
        ├──► network account    ── (same as above)
        ├──► dns account        ── (same as above)
        ├──► automation account ── (same as above)
        ├──► artifacts account  ── (same as above)
        ├──► dev account        ── (same as above)
        ├──► staging account    ── (same as above)
        └──► prod account       ── (same as above)
```

### Why This Architecture?

- **Per-Account Role**: Each account needs its own IAM role because Vanta scans resources per-account
- **No Delegated Administrator**: Unlike GuardDuty/Security Hub, Vanta does not use AWS Organizations delegation
- **No Cross-Account Dependencies**: Each account's role is independent — all accounts can be deployed in parallel
- **Management Account Extra Policy**: Only the root account needs organization-level read access for account enumeration

## Vanta Connection Types

When connecting AWS in the Vanta dashboard, there are two options:

**Individual Account** — Connect one AWS account at a time. You paste each account's role ARN into the Vanta dashboard
manually. If new accounts are added to the organization later, they must be connected manually in Vanta.

**Organization (recommended)** — Connect once using the management account. Vanta automatically discovers all accounts
in the AWS Organization and scans them. New accounts added in the future are picked up automatically.

| Aspect                      | Individual Account                      | Organization                                              |
|-----------------------------|-----------------------------------------|-----------------------------------------------------------|
| **Scope**                   | One account at a time                   | All accounts in the AWS Organization                      |
| **Auto-discovery**          | No — each account connected manually    | Yes — new accounts auto-detected                          |
| **IAM Role/Policies**       | Identical                               | Identical                                                 |
| **Management Account**      | No special requirements                 | Needs `VantaManagementAccountPermissions` for enumeration |

### What Changes on the AWS Side?

**Nothing.** Both methods require the same `vanta-auditor` IAM role in every account (the management account always
includes the extra `VantaManagementAccountPermissions` policy regardless of connection type). The only difference is
how the Vanta platform *discovers* accounts:

- **Individual**: You tell Vanta about each account by pasting the role ARN
- **Organization**: Vanta calls `organizations:ListAccounts` from the management account role to enumerate all accounts,
  then assumes the `vanta-auditor` role in each one

## Deployment Model Comparison

### Vanta vs Other AWS Security Services

| Aspect                  | Vanta Integration             | AWS GuardDuty                        | AWS Config                        |
|-------------------------|-------------------------------|--------------------------------------|-----------------------------------|
| **Purpose**             | GRC evidence collection       | Threat detection                     | Configuration compliance          |
| **Deployment Approach** | Per-account IAM role          | Delegated administrator (3 steps)    | Per-account with aggregation      |
| **Central Account**     | N/A (Vanta is external SaaS)  | Security (delegated administrator)   | Security (aggregator)             |
| **Organization-Wide**   | No (per account)              | Yes                                  | Yes (conformance packs from root) |
| **Access Type**         | Read-only (cross-account)     | Service-managed                      | Service-managed                   |

## IAM Policies

### SecurityAudit (AWS Managed)

- **ARN**: `arn:aws:iam::aws:policy/SecurityAudit`
- **Purpose**: Broad read-only access to security-relevant configurations across all AWS services
- **Maintained by**: AWS — automatically updated when new services are added
- **Scope**: Describe, Get, List actions across IAM, EC2, S3, RDS, Lambda, ECS, EKS, and many more

### VantaAdditionalPermissions (Custom)

Supplements `SecurityAudit` with Identity Center permissions and denies access to sensitive data:

| Statement                      | Effect | Purpose                        |
|--------------------------------|--------|--------------------------------|
| VantaIdentityCenterPermissions | Allow  | IAM Identity Center scanning   |
| VantaDenyDataAccess            | Deny   | Block access to sensitive data |

### VantaManagementAccountPermissions (Custom, root only)

Organization-level read access for the management account:

| Statement                         | Effect | Purpose                 |
|-----------------------------------|--------|-------------------------|
| VantaManagementAccountPermissions | Allow  | Enumerate org structure |

## Security Considerations

1. **Read-Only Access**: All permissions are read-only — Vanta cannot create, modify, or delete any AWS resources
2. **Explicit Deny**: Sensitive data actions are explicitly denied, even if `SecurityAudit` would allow them
3. **External ID**: The trust policy requires a matching external ID, preventing confused deputy attacks
4. **Scoped Principal**: The trust policy allows only Vanta's `scanner` roles from specific AWS accounts, not entire accounts
5. **CloudTrail Audit**: Every API call from Vanta is logged in CloudTrail
6. **Minimal Permissions**: The management account policy only grants organization read access, not administrative permissions
## Usage

**Stack Level**: Global

The following are example snippets for how to use this component.

### Default Configuration (`stacks/catalog/aws-vanta/defaults.yaml`)

```yaml
components:
  terraform:
    aws-vanta/defaults:
      metadata:
        type: abstract
        component: "aws-vanta"
      vars:
        enabled: true
        iam_role_name: "vanta-auditor"
        external_id: "<external-id-from-vanta-dashboard>"
        management_account_permissions_enabled: false
```

### Member Account Configuration (`stacks/catalog/aws-vanta/member-account.yaml`)

```yaml
import:
  - catalog/aws-vanta/defaults

components:
  terraform:
    aws-vanta:
      metadata:
        component: "aws-vanta"
        inherits:
          - "aws-vanta/defaults"
```

### Management Account Configuration (`stacks/catalog/aws-vanta/management-account.yaml`)

```yaml
import:
  - catalog/aws-vanta/defaults

components:
  terraform:
    aws-vanta:
      metadata:
        component: "aws-vanta"
        inherits:
          - "aws-vanta/defaults"
      vars:
        management_account_permissions_enabled: true
```

## Prerequisites

Before deploying this component:

1. **Vanta Account**: Ensure your Vanta account is active and the AWS integration is initiated in the Vanta dashboard.
2. **Vanta External ID**: Obtain the external ID from the Vanta dashboard and configure it in `defaults.yaml`.

## Provisioning

### No Dependencies — Deploy All Accounts in Parallel

Unlike other security services, Vanta has no inter-service dependencies. All accounts can be provisioned simultaneously.

```bash
# Management account:
atmos terraform apply aws-vanta -s core-gbl-root

# Core tenant:
atmos terraform apply aws-vanta -s core-gbl-security
atmos terraform apply aws-vanta -s core-gbl-audit
atmos terraform apply aws-vanta -s core-gbl-auto
atmos terraform apply aws-vanta -s core-gbl-artifacts
atmos terraform apply aws-vanta -s core-gbl-dns
atmos terraform apply aws-vanta -s core-gbl-network

# Platform tenant:
atmos terraform apply aws-vanta -s plat-gbl-sandbox
atmos terraform apply aws-vanta -s plat-gbl-dev
atmos terraform apply aws-vanta -s plat-gbl-staging
atmos terraform apply aws-vanta -s plat-gbl-prod
```

### Post-Deployment Steps

After all IAM roles are deployed:

1. **Provide Management Account Role ARN to Vanta**: Copy the `vanta_auditor_role_arn` output from the
   management account deployment and enter it in the Vanta dashboard
2. **Configure Regions in Vanta**: Select all regions where you have resources in the Vanta AWS integration settings
3. **Wait for Initial Scan**: Vanta takes up to 2 hours to complete the initial resource scan
4. **Verify Connection**: Check the Vanta dashboard for successful integration status across all accounts

## Troubleshooting

### Vanta Cannot Assume Role

**Problem**: Vanta dashboard shows "Unable to connect" or "Access denied" for an account.

**Solution**:

1. Verify the external ID in `defaults.yaml` matches the value in the Vanta dashboard
2. Check the trust policy allows Vanta's scanner roles:
   ```bash
   aws iam get-role --role-name vanta-auditor --query 'Role.AssumeRolePolicyDocument'
   ```
3. Verify both policies are attached:
   ```bash
   aws iam list-attached-role-policies --role-name vanta-auditor
   ```

### Missing Permissions in Vanta Scan

**Problem**: Vanta reports incomplete scans or missing evidence for certain services.

**Solution**:

1. Check if `VantaAdditionalPermissions` policy is attached
2. Verify `SecurityAudit` managed policy is current (AWS updates it automatically)
3. For the management account, verify `VantaManagementAccountPermissions` is attached
4. Check Vanta's help docs for any newly required permissions

### External ID Changed

**Problem**: Vanta regenerated the external ID and existing roles no longer work.

**Solution**:

1. Update `external_id` in `stacks/catalog/aws-vanta/defaults.yaml`
2. Redeploy all accounts

<!-- prettier-ignore-start -->
<!-- prettier-ignore-end -->


<!-- markdownlint-disable -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.0.0 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.0.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | >= 5.0.0 |
| <a name="provider_terraform"></a> [terraform](#provider\_terraform) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_account_map"></a> [account\_map](#module\_account\_map) | cloudposse/stack-config/yaml//modules/remote-state | 2.0.0 |
| <a name="module_this"></a> [this](#module\_this) | cloudposse/label/null | 0.25.0 |

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.vanta_additional_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy.vanta_management_account_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.vanta_auditor](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.security_audit](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.vanta_additional_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.vanta_management_account_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [terraform_data.account_verification](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/resources/data) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.vanta_additional_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.vanta_management_account_permissions](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_partition.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/partition) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_account_map"></a> [account\_map](#input\_account\_map) | Static account map configuration. Only used when `account_map_enabled` is `false`.<br/>Map keys use `tenant-stage` format (e.g., `core-security`, `core-audit`, `plat-prod`). | <pre>object({<br/>    full_account_map              = map(string)<br/>    audit_account_account_name    = optional(string, "")<br/>    root_account_account_name     = optional(string, "")<br/>    identity_account_account_name = optional(string, "")<br/>    aws_partition                 = optional(string, "aws")<br/>    iam_role_arn_templates        = optional(map(string), {})<br/>  })</pre> | <pre>{<br/>  "audit_account_account_name": "",<br/>  "aws_partition": "aws",<br/>  "full_account_map": {},<br/>  "iam_role_arn_templates": {},<br/>  "identity_account_account_name": "",<br/>  "root_account_account_name": ""<br/>}</pre> | no |
| <a name="input_account_map_component_name"></a> [account\_map\_component\_name](#input\_account\_map\_component\_name) | The name of the account-map component | `string` | `"account-map"` | no |
| <a name="input_account_map_enabled"></a> [account\_map\_enabled](#input\_account\_map\_enabled) | Enable the account map component. When true, the component fetches account mappings from the<br/>`account-map` component via remote state. When false (default), the component uses the static `account_map` variable instead. | `bool` | `false` | no |
| <a name="input_account_map_tenant"></a> [account\_map\_tenant](#input\_account\_map\_tenant) | The tenant where the `account_map` component required by remote-state is deployed | `string` | `"core"` | no |
| <a name="input_account_verification_enabled"></a> [account\_verification\_enabled](#input\_account\_verification\_enabled) | Enable account verification. When true (default), the component verifies that Terraform is executing<br/>in the correct AWS account by comparing the current account ID against the expected account from the<br/>account\_map based on the component's tenant-stage context. | `bool` | `true` | no |
| <a name="input_additional_tag_map"></a> [additional\_tag\_map](#input\_additional\_tag\_map) | Additional key-value pairs to add to each map in `tags_as_list_of_maps`. Not added to `tags` or `id`.<br/>This is for some rare cases where resources want additional configuration of tags<br/>and therefore take a list of maps with tag key, value, and additional configuration. | `map(string)` | `{}` | no |
| <a name="input_attributes"></a> [attributes](#input\_attributes) | ID element. Additional attributes (e.g. `workers` or `cluster`) to add to `id`,<br/>in the order they appear in the list. New attributes are appended to the<br/>end of the list. The elements of the list are joined by the `delimiter`<br/>and treated as a single ID element. | `list(string)` | `[]` | no |
| <a name="input_context"></a> [context](#input\_context) | Single object for setting entire context at once.<br/>See description of individual variables for details.<br/>Leave string and numeric variables as `null` to use default value.<br/>Individual variable settings (non-null) override settings in context object,<br/>except for attributes, tags, and additional\_tag\_map, which are merged. | `any` | <pre>{<br/>  "additional_tag_map": {},<br/>  "attributes": [],<br/>  "delimiter": null,<br/>  "descriptor_formats": {},<br/>  "enabled": true,<br/>  "environment": null,<br/>  "id_length_limit": null,<br/>  "label_key_case": null,<br/>  "label_order": [],<br/>  "label_value_case": null,<br/>  "labels_as_tags": [<br/>    "unset"<br/>  ],<br/>  "name": null,<br/>  "namespace": null,<br/>  "regex_replace_chars": null,<br/>  "stage": null,<br/>  "tags": {},<br/>  "tenant": null<br/>}</pre> | no |
| <a name="input_delimiter"></a> [delimiter](#input\_delimiter) | Delimiter to be used between ID elements.<br/>Defaults to `-` (hyphen). Set to `""` to use no delimiter at all. | `string` | `null` | no |
| <a name="input_descriptor_formats"></a> [descriptor\_formats](#input\_descriptor\_formats) | Describe additional descriptors to be output in the `descriptors` output map.<br/>Map of maps. Keys are names of descriptors. Values are maps of the form<br/>`{<br/>   format = string<br/>   labels = list(string)<br/>}`<br/>(Type is `any` so the map values can later be enhanced to provide additional options.)<br/>`format` is a Terraform format string to be passed to the `format()` function.<br/>`labels` is a list of labels, in order, to pass to `format()` function.<br/>Label values will be normalized before being passed to `format()` so they will be<br/>identical to how they appear in `id`.<br/>Default is `{}` (`descriptors` output will be empty). | `any` | `{}` | no |
| <a name="input_enabled"></a> [enabled](#input\_enabled) | Set to false to prevent the module from creating any resources | `bool` | `null` | no |
| <a name="input_environment"></a> [environment](#input\_environment) | ID element. Usually used for region e.g. 'uw2', 'us-west-2', OR role 'prod', 'staging', 'dev', 'UAT' | `string` | `null` | no |
| <a name="input_external_id"></a> [external\_id](#input\_external\_id) | External ID from the Vanta UI used in the IAM role trust policy to prevent confused deputy attacks.<br/>Obtain this value from the Vanta dashboard or from your Vanta consultant. | `string` | n/a | yes |
| <a name="input_global_environment"></a> [global\_environment](#input\_global\_environment) | Global environment name | `string` | `"gbl"` | no |
| <a name="input_iam_role_name"></a> [iam\_role\_name](#input\_iam\_role\_name) | Name of the IAM role created for Vanta | `string` | `"vanta-auditor"` | no |
| <a name="input_id_length_limit"></a> [id\_length\_limit](#input\_id\_length\_limit) | Limit `id` to this many characters (minimum 6).<br/>Set to `0` for unlimited length.<br/>Set to `null` for keep the existing setting, which defaults to `0`.<br/>Does not affect `id_full`. | `number` | `null` | no |
| <a name="input_label_key_case"></a> [label\_key\_case](#input\_label\_key\_case) | Controls the letter case of the `tags` keys (label names) for tags generated by this module.<br/>Does not affect keys of tags passed in via the `tags` input.<br/>Possible values: `lower`, `title`, `upper`.<br/>Default value: `title`. | `string` | `null` | no |
| <a name="input_label_order"></a> [label\_order](#input\_label\_order) | The order in which the labels (ID elements) appear in the `id`.<br/>Defaults to ["namespace", "environment", "stage", "name", "attributes"].<br/>You can omit any of the 6 labels ("tenant" is the 6th), but at least one must be present. | `list(string)` | `null` | no |
| <a name="input_label_value_case"></a> [label\_value\_case](#input\_label\_value\_case) | Controls the letter case of ID elements (labels) as included in `id`,<br/>set as tag values, and output by this module individually.<br/>Does not affect values of tags passed in via the `tags` input.<br/>Possible values: `lower`, `title`, `upper` and `none` (no transformation).<br/>Set this to `title` and set `delimiter` to `""` to yield Pascal Case IDs.<br/>Default value: `lower`. | `string` | `null` | no |
| <a name="input_labels_as_tags"></a> [labels\_as\_tags](#input\_labels\_as\_tags) | Set of labels (ID elements) to include as tags in the `tags` output.<br/>Default is to include all labels.<br/>Tags with empty values will not be included in the `tags` output.<br/>Set to `[]` to suppress all generated tags.<br/>**Notes:**<br/>  The value of the `name` tag, if included, will be the `id`, not the `name`.<br/>  Unlike other `null-label` inputs, the initial setting of `labels_as_tags` cannot be<br/>  changed in later chained modules. Attempts to change it will be silently ignored. | `set(string)` | <pre>[<br/>  "default"<br/>]</pre> | no |
| <a name="input_management_account_permissions_enabled"></a> [management\_account\_permissions\_enabled](#input\_management\_account\_permissions\_enabled) | Enable management account permissions. When true, attaches VantaManagementAccountPermissions policy<br/>to the role for organization-level read access. Only enable for the management (root) account. | `bool` | `false` | no |
| <a name="input_name"></a> [name](#input\_name) | ID element. Usually the component or solution name, e.g. 'app' or 'jenkins'.<br/>This is the only ID element not also included as a `tag`.<br/>The "name" tag is set to the full `id` string. There is no tag with the value of the `name` input. | `string` | `null` | no |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | ID element. Usually an abbreviation of your organization name, e.g. 'eg' or 'cp', to help ensure generated IDs are globally unique | `string` | `null` | no |
| <a name="input_privileged"></a> [privileged](#input\_privileged) | true if the default provider already has access to the backend | `bool` | `false` | no |
| <a name="input_regex_replace_chars"></a> [regex\_replace\_chars](#input\_regex\_replace\_chars) | Terraform regular expression (regex) string.<br/>Characters matching the regex will be removed from the ID elements.<br/>If not set, `"/[^a-zA-Z0-9-]/"` is used to remove all characters other than hyphens, letters and digits. | `string` | `null` | no |
| <a name="input_region"></a> [region](#input\_region) | AWS Region | `string` | n/a | yes |
| <a name="input_root_account_stage"></a> [root\_account\_stage](#input\_root\_account\_stage) | The stage name for the Organization root (management) account. This is used to lookup account IDs from account names<br/>using the `account-map` component. | `string` | `"root"` | no |
| <a name="input_stage"></a> [stage](#input\_stage) | ID element. Usually used to indicate role, e.g. 'prod', 'staging', 'source', 'build', 'test', 'deploy', 'release' | `string` | `null` | no |
| <a name="input_tags"></a> [tags](#input\_tags) | Additional tags (e.g. `{'BusinessUnit': 'XYZ'}`).<br/>Neither the tag keys nor the tag values will be modified by this module. | `map(string)` | `{}` | no |
| <a name="input_tenant"></a> [tenant](#input\_tenant) | ID element \_(Rarely used, not included by default)\_. A customer identifier, indicating who this instance of a resource is for | `string` | `null` | no |
| <a name="input_vanta_account_ids"></a> [vanta\_account\_ids](#input\_vanta\_account\_ids) | List of Vanta's AWS account IDs used in the IAM role trust policy for cross-account access.<br/>Vanta operates from multiple AWS accounts across regions. All three must be trusted for full<br/>multi-region support. See: https://help.vanta.com/en/articles/11345698-porting-aws-integrations-across-regions | `list(string)` | <pre>[<br/>  "956993596390",<br/>  "850507053895",<br/>  "654654195764"<br/>]</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_vanta_additional_permissions_policy_arn"></a> [vanta\_additional\_permissions\_policy\_arn](#output\_vanta\_additional\_permissions\_policy\_arn) | ARN of the VantaAdditionalPermissions IAM policy |
| <a name="output_vanta_auditor_role_arn"></a> [vanta\_auditor\_role\_arn](#output\_vanta\_auditor\_role\_arn) | ARN of the Vanta auditor IAM role |
| <a name="output_vanta_auditor_role_name"></a> [vanta\_auditor\_role\_name](#output\_vanta\_auditor\_role\_name) | Name of the Vanta auditor IAM role |
| <a name="output_vanta_management_account_permissions_policy_arn"></a> [vanta\_management\_account\_permissions\_policy\_arn](#output\_vanta\_management\_account\_permissions\_policy\_arn) | ARN of the VantaManagementAccountPermissions IAM policy (management account only) |
<!-- markdownlint-restore -->



## References


- [Connecting Vanta to an AWS Organization](https://help.vanta.com/en/articles/11345628-connecting-vanta-aws-organization) - Official Vanta documentation for connecting an AWS Organization

- [Connecting AWS with CloudFormation](https://help.vanta.com/en/articles/11345623-connecting-aws-with-cloudformation) - Vanta documentation for connecting AWS with CloudFormation

- [VantaAdditionalPermissions Policy Details](https://help.vanta.com/en/articles/11345613-modifying-the-vantaadditionalpermission-policy-for-individual-aws-accounts) - Vanta documentation for modifying the VantaAdditionalPermission policy

- [AWS SecurityAudit Policy](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/SecurityAudit.html) - AWS managed policy reference for SecurityAudit

- [cloudposse-terraform-components](https://github.com/orgs/cloudposse-terraform-components/repositories) - Cloud Posse's upstream component




[<img src="https://cloudposse.com/logo-300x69.svg" height="32" align="right"/>](https://cpco.io/homepage?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-vanta&utm_content=)

