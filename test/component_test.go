package test

import (
	"context"
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

func (s *ComponentSuite) TestBasic() {
	const component = "aws-vanta/basic"

	defer s.DestroyAtmosComponent(s.T(), component, testStack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, testStack, nil)
	assert.NotNil(s.T(), options)

	roleArn := atmos.Output(s.T(), options, "vanta_auditor_role_arn")
	assert.NotEmpty(s.T(), roleArn)

	roleName := atmos.Output(s.T(), options, "vanta_auditor_role_name")
	assert.Equal(s.T(), "vanta-auditor", roleName)

	additionalPolicyArn := atmos.Output(s.T(), options, "vanta_additional_permissions_policy_arn")
	assert.NotEmpty(s.T(), additionalPolicyArn)

	managementPolicyArn := atmos.Output(s.T(), options, "vanta_management_account_permissions_policy_arn")
	assert.Empty(s.T(), managementPolicyArn)

	// Verify the IAM role exists and has the correct trust policy
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

	policyArns := make([]string, len(policies.AttachedPolicies))
	for i, p := range policies.AttachedPolicies {
		policyArns[i] = *p.PolicyArn
	}
	assert.Contains(s.T(), policyArns, additionalPolicyArn)

	s.DriftTest(component, testStack, nil)
}

func (s *ComponentSuite) TestManagementAccount() {
	const component = "aws-vanta/management-account"

	defer s.DestroyAtmosComponent(s.T(), component, testStack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, testStack, nil)
	assert.NotNil(s.T(), options)

	roleArn := atmos.Output(s.T(), options, "vanta_auditor_role_arn")
	assert.NotEmpty(s.T(), roleArn)

	roleName := atmos.Output(s.T(), options, "vanta_auditor_role_name")
	assert.Equal(s.T(), "vanta-auditor", roleName)

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

	s.DriftTest(component, testStack, nil)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "aws-vanta/disabled"

	s.VerifyEnabledFlag(component, testStack, nil)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	helper.Run(t, suite)
}
