// GENERATED, DO NOT EDIT THIS FILE
package aws

import (
	"github.com/cloudskiff/driftctl/pkg/resource"
	"github.com/zclconf/go-cty/cty"
)

const AwsIamUserPolicyResourceType = "aws_iam_user_policy"

type AwsIamUserPolicy struct {
	Id         string     `cty:"id" computed:"true"`
	Name       *string    `cty:"name" computed:"true"`
	NamePrefix *string    `cty:"name_prefix"`
	Policy     *string    `cty:"policy" jsonstring:"true"`
	User       *string    `cty:"user"`
	CtyVal     *cty.Value `diff:"-"`
}

func (r *AwsIamUserPolicy) TerraformId() string {
	return r.Id
}

func (r *AwsIamUserPolicy) TerraformType() string {
	return AwsIamUserPolicyResourceType
}

func (r *AwsIamUserPolicy) CtyValue() *cty.Value {
	return r.CtyVal
}

func initAwsIAMUserPolicyMetaData(resourceSchemaRepository resource.SchemaRepositoryInterface) {
	resourceSchemaRepository.UpdateSchema(AwsIamUserPolicyResourceType, map[string]func(attributeSchema *resource.AttributeSchema){
		"policy": func(attributeSchema *resource.AttributeSchema) {
			attributeSchema.JsonString = true
		},
	})
}
