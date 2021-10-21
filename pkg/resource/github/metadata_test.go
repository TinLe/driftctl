package github

import (
	"testing"

	"github.com/cloudskiff/driftctl/pkg/resource"
	tf "github.com/cloudskiff/driftctl/pkg/terraform"
	testresource "github.com/cloudskiff/driftctl/test/resource"
	"github.com/stretchr/testify/assert"
)

func TestGitHub_Metadata_Flags(t *testing.T) {
	testcases := map[string][]resource.Flags{
		GithubBranchProtectionResourceType: {resource.FlagDeepMode},
		GithubMembershipResourceType:       {resource.FlagDeepMode},
		GithubTeamMembershipResourceType:   {resource.FlagDeepMode},
		GithubRepositoryResourceType:       {resource.FlagDeepMode},
		GithubTeamResourceType:             {resource.FlagDeepMode},
	}

	schemaRepository := testresource.InitFakeSchemaRepository(tf.GITHUB, "4.4.0")
	InitResourcesMetadata(schemaRepository)

	for ty, flags := range testcases {
		t.Run(ty, func(tt *testing.T) {
			sch, exist := schemaRepository.GetSchema(ty)
			assert.True(tt, exist)

			if len(flags) == 0 {
				assert.Equal(tt, resource.Flags(0x0), sch.Flags, "should not have any flag")
				return
			}

			for _, flag := range flags {
				assert.Truef(tt, sch.Flags.HasFlag(flag), "should have given flag %d", flag)
			}
		})
	}
}
