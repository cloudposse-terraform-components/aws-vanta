locals {
  enabled     = module.this.enabled
  tags        = module.this.tags
  account_map = module.account_map.outputs.full_account_map

  account_id         = one(data.aws_caller_identity.current[*].account_id)
  current_account_id = local.account_id

  partition = one(data.aws_partition.current[*].partition)

  management_account_permissions_enabled = local.enabled && var.management_account_permissions_enabled
}

data "aws_caller_identity" "current" {
  count = local.enabled ? 1 : 0
}

data "aws_partition" "current" {
  count = local.enabled ? 1 : 0
}

# Trust policy: allow Vanta's AWS accounts to assume the role with external ID.
# Vanta operates from multiple AWS accounts across regions — all must be trusted.
# See: https://help.vanta.com/en/articles/11345698-porting-aws-integrations-across-regions
data "aws_iam_policy_document" "assume_role" {
  count = local.enabled ? 1 : 0

  statement {
    sid     = "VantaAssumeRole"
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = [for id in var.vanta_account_ids : "arn:${local.partition}:iam::${id}:role/scanner"]
    }

    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = [var.external_id]
    }
  }
}

# IAM role for Vanta auditor
resource "aws_iam_role" "vanta_auditor" {
  count = local.enabled ? 1 : 0

  name               = var.iam_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_role[0].json
  tags               = local.tags
}

# Attach AWS managed SecurityAudit policy
resource "aws_iam_role_policy_attachment" "security_audit" {
  count = local.enabled ? 1 : 0

  role       = aws_iam_role.vanta_auditor[0].name
  policy_arn = "arn:${local.partition}:iam::aws:policy/SecurityAudit"
}

# VantaAdditionalPermissions policy document
data "aws_iam_policy_document" "vanta_additional_permissions" {
  count = local.enabled ? 1 : 0

  # IAM Identity Center permissions for identity scanning
  statement {
    sid    = "VantaIdentityCenterPermissions"
    effect = "Allow"
    actions = [
      "identitystore:DescribeGroup",
      "identitystore:DescribeGroupMembership",
      "identitystore:DescribeUser",
      "identitystore:GetGroupId",
      "identitystore:GetGroupMembershipId",
      "identitystore:GetUserId",
      "identitystore:IsMemberInGroups",
      "identitystore:ListGroupMemberships",
      "identitystore:ListGroupMembershipsForMember",
      "identitystore:ListGroups",
      "identitystore:ListUsers",
    ]
    resources = ["*"]
  }

  # Deny access to sensitive data
  statement {
    sid    = "VantaDenyDataAccess"
    effect = "Deny"
    actions = [
      "datapipeline:EvaluateExpression",
      "datapipeline:QueryObjects",
      "rds:DownloadDBLogFilePortion",
    ]
    resources = ["*"]
  }
}

# Create the VantaAdditionalPermissions managed policy
resource "aws_iam_policy" "vanta_additional_permissions" {
  count = local.enabled ? 1 : 0

  name        = "VantaAdditionalPermissions"
  description = "Additional read-only permissions for Vanta auditor beyond SecurityAudit"
  policy      = data.aws_iam_policy_document.vanta_additional_permissions[0].json
  tags        = local.tags
}

# Attach VantaAdditionalPermissions to the role
resource "aws_iam_role_policy_attachment" "vanta_additional_permissions" {
  count = local.enabled ? 1 : 0

  role       = aws_iam_role.vanta_auditor[0].name
  policy_arn = aws_iam_policy.vanta_additional_permissions[0].arn
}

# Management account permissions (organization-level reads)
data "aws_iam_policy_document" "vanta_management_account_permissions" {
  count = local.management_account_permissions_enabled ? 1 : 0

  statement {
    sid    = "VantaManagementAccountPermissions"
    effect = "Allow"
    actions = [
      "organizations:ListAccounts",
      "organizations:ListAccountsForParent",
      "organizations:DescribeOrganization",
      "organizations:DescribeOrganizationalUnit",
      "organizations:ListOrganizationalUnitsForParent",
      "organizations:ListRoots",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "vanta_management_account_permissions" {
  count = local.management_account_permissions_enabled ? 1 : 0

  name        = "VantaManagementAccountPermissions"
  description = "Organization-level read permissions for Vanta auditor in the management account"
  policy      = data.aws_iam_policy_document.vanta_management_account_permissions[0].json
  tags        = local.tags
}

resource "aws_iam_role_policy_attachment" "vanta_management_account_permissions" {
  count = local.management_account_permissions_enabled ? 1 : 0

  role       = aws_iam_role.vanta_auditor[0].name
  policy_arn = aws_iam_policy.vanta_management_account_permissions[0].arn
}
