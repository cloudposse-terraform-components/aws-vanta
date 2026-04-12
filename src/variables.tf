variable "account_verification_enabled" {
  type        = bool
  description = <<-DOC
  Enable account verification. When true (default), the component verifies that Terraform is executing
  in the correct AWS account by comparing the current account ID against the expected account from the
  account_map based on the component's tenant-stage context.
  DOC
  default     = true
}

variable "account_map_enabled" {
  type        = bool
  description = <<-DOC
  Enable the account map component. When true, the component fetches account mappings from the
  `account-map` component via remote state. When false (default), the component uses the static `account_map` variable instead.
  DOC
  default     = false
}

variable "account_map" {
  type = object({
    full_account_map              = map(string)
    audit_account_account_name    = optional(string, "")
    root_account_account_name     = optional(string, "")
    identity_account_account_name = optional(string, "")
    aws_partition                 = optional(string, "aws")
    iam_role_arn_templates        = optional(map(string), {})
  })
  description = <<-DOC
  Static account map configuration. Only used when `account_map_enabled` is `false`.
  Map keys use `tenant-stage` format (e.g., `core-security`, `core-audit`, `plat-prod`).
  DOC
  default = {
    full_account_map              = {}
    audit_account_account_name    = ""
    root_account_account_name     = ""
    identity_account_account_name = ""
    aws_partition                 = "aws"
    iam_role_arn_templates        = {}
  }
}

variable "account_map_component_name" {
  type        = string
  description = "The name of the account-map component"
  default     = "account-map"
}

variable "account_map_tenant" {
  type        = string
  default     = "core"
  description = "The tenant where the `account_map` component required by remote-state is deployed"
}

variable "global_environment" {
  type        = string
  default     = "gbl"
  description = "Global environment name"
}

variable "privileged" {
  type        = bool
  default     = false
  description = "true if the default provider already has access to the backend"
}

variable "region" {
  type        = string
  description = "AWS Region"
}

variable "root_account_stage" {
  type        = string
  default     = "root"
  description = <<-DOC
  The stage name for the Organization root (management) account. This is used to lookup account IDs from account names
  using the `account-map` component.
  DOC
}

variable "vanta_account_ids" {
  type        = list(string)
  description = <<-DOC
  List of Vanta's AWS account IDs used in the IAM role trust policy for cross-account access.
  Vanta operates from multiple AWS accounts across regions. All three must be trusted for full
  multi-region support. See: https://help.vanta.com/en/articles/11345698-porting-aws-integrations-across-regions
  DOC
  default = [
    "956993596390",
    "850507053895",
    "654654195764",
  ]

  validation {
    condition     = length(var.vanta_account_ids) > 0
    error_message = "vanta_account_ids must contain at least one AWS account ID."
  }

  validation {
    condition     = alltrue([for id in var.vanta_account_ids : can(regex("^[0-9]{12}$", id))])
    error_message = "All vanta_account_ids must be valid 12-digit AWS account IDs."
  }
}

variable "external_id" {
  type        = string
  description = <<-DOC
  External ID from the Vanta UI used in the IAM role trust policy to prevent confused deputy attacks.
  Obtain this value from the Vanta dashboard or from your Vanta consultant.
  DOC
  sensitive   = true

  validation {
    condition     = length(var.external_id) > 0
    error_message = "external_id must not be empty. This value is required for confused deputy protection in the IAM trust policy."
  }
}

variable "iam_role_name" {
  type        = string
  description = "Name of the IAM role created for Vanta"
  default     = "vanta-auditor"
}

variable "management_account_permissions_enabled" {
  type        = bool
  description = <<-DOC
  Enable management account permissions. When true, attaches VantaManagementAccountPermissions policy
  to the role for organization-level read access. Only enable for the management (root) account.
  DOC
  default     = false
}
