package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryFileDataSource_Metadata(t *testing.T) {
	ds := NewRepositoryFileDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository_file", resp.TypeName)
}

func TestRepositoryFileDataSource_Schema(t *testing.T) {
	ds := NewRepositoryFileDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Get information on a file in a GitHub repository")

	// Check optional attributes (repository and full_name are mutually exclusive)
	repositoryAttr, ok := resp.Schema.Attributes["repository"]
	assert.True(t, ok)
	assert.True(t, repositoryAttr.IsOptional())

	fullNameAttr, ok := resp.Schema.Attributes["full_name"]
	assert.True(t, ok)
	assert.True(t, fullNameAttr.IsOptional())

	branchAttr, ok := resp.Schema.Attributes["branch"]
	assert.True(t, ok)
	assert.True(t, branchAttr.IsOptional())

	// Check required attribute
	fileAttr, ok := resp.Schema.Attributes["file"]
	assert.True(t, ok)
	assert.True(t, fileAttr.IsRequired())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	refAttr, ok := resp.Schema.Attributes["ref"]
	assert.True(t, ok)
	assert.True(t, refAttr.IsComputed())

	contentAttr, ok := resp.Schema.Attributes["content"]
	assert.True(t, ok)
	assert.True(t, contentAttr.IsComputed())

	shaAttr, ok := resp.Schema.Attributes["sha"]
	assert.True(t, ok)
	assert.True(t, shaAttr.IsComputed())

	commitSHAAttr, ok := resp.Schema.Attributes["commit_sha"]
	assert.True(t, ok)
	assert.True(t, commitSHAAttr.IsComputed())

	commitMessageAttr, ok := resp.Schema.Attributes["commit_message"]
	assert.True(t, ok)
	assert.True(t, commitMessageAttr.IsComputed())

	commitAuthorAttr, ok := resp.Schema.Attributes["commit_author"]
	assert.True(t, ok)
	assert.True(t, commitAuthorAttr.IsComputed())

	commitEmailAttr, ok := resp.Schema.Attributes["commit_email"]
	assert.True(t, ok)
	assert.True(t, commitEmailAttr.IsComputed())
}

func TestRepositoryFileDataSource_Configure(t *testing.T) {
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
			errorContains: "Unexpected Data Source Configure Type",
		},
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &repositoryFileDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &datasource.ConfigureResponse{}

			ds.Configure(t.Context(), req, resp)

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
						assert.Equal(t, clientData.Client, ds.client)
						assert.Equal(t, clientData.Owner, ds.owner)
					}
				}
			}
		})
	}
}

// Note: Tests for Read() method that require GitHub API calls should be
// implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
