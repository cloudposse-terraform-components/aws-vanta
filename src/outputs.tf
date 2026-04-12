output "vanta_auditor_role_arn" {
  description = "ARN of the Vanta auditor IAM role"
  value       = try(aws_iam_role.vanta_auditor[0].arn, "")
}

output "vanta_auditor_role_name" {
  description = "Name of the Vanta auditor IAM role"
  value       = try(aws_iam_role.vanta_auditor[0].name, "")
}

output "vanta_additional_permissions_policy_arn" {
  description = "ARN of the VantaAdditionalPermissions IAM policy"
  value       = try(aws_iam_policy.vanta_additional_permissions[0].arn, "")
}

output "vanta_management_account_permissions_policy_arn" {
  description = "ARN of the VantaManagementAccountPermissions IAM policy (management account only)"
  value       = try(aws_iam_policy.vanta_management_account_permissions[0].arn, "")
}
