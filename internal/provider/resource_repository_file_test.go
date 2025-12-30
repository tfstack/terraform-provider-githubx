package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryFileResource_Metadata(t *testing.T) {
	r := NewRepositoryFileResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository_file", resp.TypeName)
}

func TestRepositoryFileResource_Schema(t *testing.T) {
	r := NewRepositoryFileResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a file in a GitHub repository")

	// Check required attributes
	repositoryAttr, ok := resp.Schema.Attributes["repository"]
	assert.True(t, ok)
	assert.True(t, repositoryAttr.IsRequired())

	fileAttr, ok := resp.Schema.Attributes["file"]
	assert.True(t, ok)
	assert.True(t, fileAttr.IsRequired())

	contentAttr, ok := resp.Schema.Attributes["content"]
	assert.True(t, ok)
	assert.True(t, contentAttr.IsRequired())

	// Check optional attributes
	branchAttr, ok := resp.Schema.Attributes["branch"]
	assert.True(t, ok)
	assert.True(t, branchAttr.IsOptional())

	commitMessageAttr, ok := resp.Schema.Attributes["commit_message"]
	assert.True(t, ok)
	assert.True(t, commitMessageAttr.IsOptional())
	assert.True(t, commitMessageAttr.IsComputed())

	commitAuthorAttr, ok := resp.Schema.Attributes["commit_author"]
	assert.True(t, ok)
	assert.True(t, commitAuthorAttr.IsOptional())

	commitEmailAttr, ok := resp.Schema.Attributes["commit_email"]
	assert.True(t, ok)
	assert.True(t, commitEmailAttr.IsOptional())

	overwriteOnCreateAttr, ok := resp.Schema.Attributes["overwrite_on_create"]
	assert.True(t, ok)
	assert.True(t, overwriteOnCreateAttr.IsOptional())
	assert.True(t, overwriteOnCreateAttr.IsComputed())

	autocreateBranchAttr, ok := resp.Schema.Attributes["autocreate_branch"]
	assert.True(t, ok)
	assert.True(t, autocreateBranchAttr.IsOptional())
	assert.True(t, autocreateBranchAttr.IsComputed())

	autocreateBranchSourceAttr, ok := resp.Schema.Attributes["autocreate_branch_source_branch"]
	assert.True(t, ok)
	assert.True(t, autocreateBranchSourceAttr.IsOptional())
	assert.True(t, autocreateBranchSourceAttr.IsComputed())

	autocreateBranchSourceSHAAttr, ok := resp.Schema.Attributes["autocreate_branch_source_sha"]
	assert.True(t, ok)
	assert.True(t, autocreateBranchSourceSHAAttr.IsOptional())
	assert.True(t, autocreateBranchSourceSHAAttr.IsComputed())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	refAttr, ok := resp.Schema.Attributes["ref"]
	assert.True(t, ok)
	assert.True(t, refAttr.IsComputed())

	shaAttr, ok := resp.Schema.Attributes["sha"]
	assert.True(t, ok)
	assert.True(t, shaAttr.IsComputed())

	commitSHAAttr, ok := resp.Schema.Attributes["commit_sha"]
	assert.True(t, ok)
	assert.True(t, commitSHAAttr.IsComputed())
}

func TestRepositoryFileResource_Configure(t *testing.T) {
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
			rs := &repositoryFileResource{}
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
