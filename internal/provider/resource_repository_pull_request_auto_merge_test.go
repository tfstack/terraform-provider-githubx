package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryPullRequestAutoMergeResource_Metadata(t *testing.T) {
	r := NewRepositoryPullRequestAutoMergeResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository_pull_request_auto_merge", resp.TypeName)
}

func TestRepositoryPullRequestAutoMergeResource_Schema(t *testing.T) {
	r := NewRepositoryPullRequestAutoMergeResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a GitHub pull request")

	// Check required attributes
	repositoryAttr, ok := resp.Schema.Attributes["repository"]
	assert.True(t, ok)
	assert.True(t, repositoryAttr.IsRequired())

	baseRefAttr, ok := resp.Schema.Attributes["base_ref"]
	assert.True(t, ok)
	assert.True(t, baseRefAttr.IsRequired())

	headRefAttr, ok := resp.Schema.Attributes["head_ref"]
	assert.True(t, ok)
	assert.True(t, headRefAttr.IsRequired())

	titleAttr, ok := resp.Schema.Attributes["title"]
	assert.True(t, ok)
	assert.True(t, titleAttr.IsRequired())

	// Check optional attributes
	bodyAttr, ok := resp.Schema.Attributes["body"]
	assert.True(t, ok)
	assert.True(t, bodyAttr.IsOptional())

	mergeWhenReadyAttr, ok := resp.Schema.Attributes["merge_when_ready"]
	assert.True(t, ok)
	assert.True(t, mergeWhenReadyAttr.IsOptional())
	assert.True(t, mergeWhenReadyAttr.IsComputed())

	mergeMethodAttr, ok := resp.Schema.Attributes["merge_method"]
	assert.True(t, ok)
	assert.True(t, mergeMethodAttr.IsOptional())
	assert.True(t, mergeMethodAttr.IsComputed())

	waitForChecksAttr, ok := resp.Schema.Attributes["wait_for_checks"]
	assert.True(t, ok)
	assert.True(t, waitForChecksAttr.IsOptional())
	assert.True(t, waitForChecksAttr.IsComputed())

	autoDeleteBranchAttr, ok := resp.Schema.Attributes["auto_delete_branch"]
	assert.True(t, ok)
	assert.True(t, autoDeleteBranchAttr.IsOptional())
	assert.True(t, autoDeleteBranchAttr.IsComputed())

	maintainerCanModifyAttr, ok := resp.Schema.Attributes["maintainer_can_modify"]
	assert.True(t, ok)
	assert.True(t, maintainerCanModifyAttr.IsOptional())
	assert.True(t, maintainerCanModifyAttr.IsComputed())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	numberAttr, ok := resp.Schema.Attributes["number"]
	assert.True(t, ok)
	assert.True(t, numberAttr.IsComputed())

	stateAttr, ok := resp.Schema.Attributes["state"]
	assert.True(t, ok)
	assert.True(t, stateAttr.IsComputed())

	mergedAttr, ok := resp.Schema.Attributes["merged"]
	assert.True(t, ok)
	assert.True(t, mergedAttr.IsComputed())

	mergedAtAttr, ok := resp.Schema.Attributes["merged_at"]
	assert.True(t, ok)
	assert.True(t, mergedAtAttr.IsComputed())

	mergeCommitSHAAttr, ok := resp.Schema.Attributes["merge_commit_sha"]
	assert.True(t, ok)
	assert.True(t, mergeCommitSHAAttr.IsComputed())

	baseSHAAttr, ok := resp.Schema.Attributes["base_sha"]
	assert.True(t, ok)
	assert.True(t, baseSHAAttr.IsComputed())

	headSHAAttr, ok := resp.Schema.Attributes["head_sha"]
	assert.True(t, ok)
	assert.True(t, headSHAAttr.IsComputed())
}

func TestRepositoryPullRequestAutoMergeResource_Configure(t *testing.T) {
	tests := []struct {
		name          string
		providerData  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name: "valid githubxClientData",
			providerData: githubxClientData{
				Client: github.NewClient(nil),
				Owner:  "test-owner",
			},
			expectError: false,
		},
		{
			name:          "invalid provider data type",
			providerData:  "invalid",
			expectError:   true,
			errorContains: "Unexpected Resource Configure Type",
		},
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &repositoryPullRequestAutoMergeResource{}
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			rs.Configure(t.Context(), req, resp)

			if tt.expectError {
				assert.True(t, resp.Diagnostics.HasError())
				if tt.errorContains != "" {
					assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.errorContains)
				}
			} else {
				assert.False(t, resp.Diagnostics.HasError())
				// If provider data is valid, verify client and owner are set
				if tt.providerData != nil {
					clientData, ok := tt.providerData.(githubxClientData)
					if ok {
						assert.Equal(t, clientData.Client, rs.client)
						assert.Equal(t, clientData.Owner, rs.owner)
					}
				}
			}
		})
	}
}

// Note: Tests for Create(), Read(), Update(), and Delete() methods that require GitHub API calls
// should be implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
