package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ComponentSuite struct {
	helper.TestSuite
}

const (
	testStack     = "default-test"
	testAwsRegion = "us-east-2"
)

// uniqueVars returns additionalVars with a unique iam_role_name per test case.
// This prevents IAM name collisions when Terraform and OpenTofu test suites
// run in parallel in the same AWS account.
func (s *ComponentSuite) uniqueVars() *map[string]interface{} {
	roleName := fmt.Sprintf("vanta-auditor-%s", s.Config.RandomIdentifier)
	vars := map[string]interface{}{
		"iam_role_name": roleName,
	}
	return &vars
}

func (s *ComponentSuite) TestBasic() {
	const component = "aws-vanta/basic"

	vars := s.uniqueVars()

	defer s.DestroyAtmosComponent(s.T(), component, testStack, vars)
	options, _ := s.DeployAtmosComponent(s.T(), component, testStack, vars)
	assert.NotNil(s.T(), options)

	roleArn := atmos.Output(s.T(), options, "vanta_auditor_role_arn")
	assert.NotEmpty(s.T(), roleArn)

	roleName := atmos.Output(s.T(), options, "vanta_auditor_role_name")
	assert.Contains(s.T(), roleName, "vanta-auditor-")

	additionalPolicyArn := atmos.Output(s.T(), options, "vanta_additional_permissions_policy_arn")
	assert.NotEmpty(s.T(), additionalPolicyArn)

	// Management account policy should be empty for basic (member account) deployment
	managementPolicyArn := atmos.Output(s.T(), options, "vanta_management_account_permissions_policy_arn")
	assert.Empty(s.T(), managementPolicyArn)

	// Verify the IAM role exists
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(testAwsRegion))
	require.NoError(s.T(), err)
	iamClient := iam.NewFromConfig(cfg)

	roleOutput, err := iamClient.GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: &roleName,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), roleArn, *roleOutput.Role.Arn)

	// Verify attached policies
	policies, err := iamClient.ListAttachedRolePolicies(context.Background(), &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	})
	require.NoError(s.T(), err)

	// Verify exactly 2 policies attached (SecurityAudit + additional permissions)
	assert.Equal(s.T(), 2, len(policies.AttachedPolicies))

	policyArns := make([]string, len(policies.AttachedPolicies))
	for i, p := range policies.AttachedPolicies {
		policyArns[i] = *p.PolicyArn
	}
	assert.Contains(s.T(), policyArns, additionalPolicyArn)

	s.DriftTest(component, testStack, vars)
}

func (s *ComponentSuite) TestManagementAccount() {
	const component = "aws-vanta/management-account"

	vars := s.uniqueVars()

	defer s.DestroyAtmosComponent(s.T(), component, testStack, vars)
	options, _ := s.DeployAtmosComponent(s.T(), component, testStack, vars)
	assert.NotNil(s.T(), options)

	roleArn := atmos.Output(s.T(), options, "vanta_auditor_role_arn")
	assert.NotEmpty(s.T(), roleArn)

	roleName := atmos.Output(s.T(), options, "vanta_auditor_role_name")
	assert.Contains(s.T(), roleName, "vanta-auditor-")

	additionalPolicyArn := atmos.Output(s.T(), options, "vanta_additional_permissions_policy_arn")
	assert.NotEmpty(s.T(), additionalPolicyArn)

	managementPolicyArn := atmos.Output(s.T(), options, "vanta_management_account_permissions_policy_arn")
	assert.NotEmpty(s.T(), managementPolicyArn)

	// Verify the IAM role exists
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(testAwsRegion))
	require.NoError(s.T(), err)
	iamClient := iam.NewFromConfig(cfg)

	roleOutput, err := iamClient.GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: &roleName,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), roleArn, *roleOutput.Role.Arn)

	// Verify all three policies are attached (SecurityAudit + Additional + Management)
	policies, err := iamClient.ListAttachedRolePolicies(context.Background(), &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, len(policies.AttachedPolicies))

	policyArns := make([]string, len(policies.AttachedPolicies))
	for i, p := range policies.AttachedPolicies {
		policyArns[i] = *p.PolicyArn
	}
	assert.Contains(s.T(), policyArns, additionalPolicyArn)
	assert.Contains(s.T(), policyArns, managementPolicyArn)

	s.DriftTest(component, testStack, vars)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "aws-vanta/disabled"

	s.VerifyEnabledFlag(component, testStack, nil)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	helper.Run(t, suite)
}
