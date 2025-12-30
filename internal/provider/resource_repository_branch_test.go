package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryBranchResource_Metadata(t *testing.T) {
	r := NewRepositoryBranchResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository_branch", resp.TypeName)
}

func TestRepositoryBranchResource_Schema(t *testing.T) {
	r := NewRepositoryBranchResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a GitHub repository branch")

	// Check required attributes
	repositoryAttr, ok := resp.Schema.Attributes["repository"]
	assert.True(t, ok)
	assert.True(t, repositoryAttr.IsRequired())

	branchAttr, ok := resp.Schema.Attributes["branch"]
	assert.True(t, ok)
	assert.True(t, branchAttr.IsRequired())

	// Check optional attributes
	sourceBranchAttr, ok := resp.Schema.Attributes["source_branch"]
	assert.True(t, ok)
	assert.True(t, sourceBranchAttr.IsOptional())
	assert.True(t, sourceBranchAttr.IsComputed())

	sourceSHAAttr, ok := resp.Schema.Attributes["source_sha"]
	assert.True(t, ok)
	assert.True(t, sourceSHAAttr.IsOptional())
	assert.True(t, sourceSHAAttr.IsComputed())

	etagAttr, ok := resp.Schema.Attributes["etag"]
	assert.True(t, ok)
	assert.True(t, etagAttr.IsOptional())
	assert.True(t, etagAttr.IsComputed())

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
}

func TestRepositoryBranchResource_Configure(t *testing.T) {
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
			rs := &repositoryBranchResource{}
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
