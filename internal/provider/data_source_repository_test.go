package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryDataSource_Metadata(t *testing.T) {
	ds := NewRepositoryDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository", resp.TypeName)
}

func TestRepositoryDataSource_Schema(t *testing.T) {
	ds := NewRepositoryDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Get information on a GitHub repository")

	// Check optional attributes (full_name and name are mutually exclusive)
	fullNameAttr, ok := resp.Schema.Attributes["full_name"]
	assert.True(t, ok)
	assert.True(t, fullNameAttr.IsOptional())
	assert.True(t, fullNameAttr.IsComputed())

	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsOptional())
	assert.True(t, nameAttr.IsComputed())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	descriptionAttr, ok := resp.Schema.Attributes["description"]
	assert.True(t, ok)
	assert.True(t, descriptionAttr.IsComputed())

	privateAttr, ok := resp.Schema.Attributes["private"]
	assert.True(t, ok)
	assert.True(t, privateAttr.IsComputed())

	visibilityAttr, ok := resp.Schema.Attributes["visibility"]
	assert.True(t, ok)
	assert.True(t, visibilityAttr.IsComputed())

	defaultBranchAttr, ok := resp.Schema.Attributes["default_branch"]
	assert.True(t, ok)
	assert.True(t, defaultBranchAttr.IsComputed())

	htmlURLAttr, ok := resp.Schema.Attributes["html_url"]
	assert.True(t, ok)
	assert.True(t, htmlURLAttr.IsComputed())

	nodeIDAttr, ok := resp.Schema.Attributes["node_id"]
	assert.True(t, ok)
	assert.True(t, nodeIDAttr.IsComputed())

	repoIDAttr, ok := resp.Schema.Attributes["repo_id"]
	assert.True(t, ok)
	assert.True(t, repoIDAttr.IsComputed())

	// Check nested attributes
	pagesAttr, ok := resp.Schema.Attributes["pages"]
	assert.True(t, ok)
	assert.True(t, pagesAttr.IsComputed())

	repositoryLicenseAttr, ok := resp.Schema.Attributes["repository_license"]
	assert.True(t, ok)
	assert.True(t, repositoryLicenseAttr.IsComputed())

	templateAttr, ok := resp.Schema.Attributes["template"]
	assert.True(t, ok)
	assert.True(t, templateAttr.IsComputed())

	topicsAttr, ok := resp.Schema.Attributes["topics"]
	assert.True(t, ok)
	assert.True(t, topicsAttr.IsComputed())
}

func TestRepositoryDataSource_Configure(t *testing.T) {
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
			ds := &repositoryDataSource{}
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

func TestSplitRepoFullName(t *testing.T) {
	tests := []struct {
		name        string
		fullName    string
		expectOwner string
		expectRepo  string
		expectError bool
	}{
		{
			name:        "valid full name",
			fullName:    "owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
			expectError: false,
		},
		{
			name:        "valid full name with dash",
			fullName:    "my-org/my-repo",
			expectOwner: "my-org",
			expectRepo:  "my-repo",
			expectError: false,
		},
		{
			name:        "invalid format - no slash",
			fullName:    "invalid",
			expectError: true,
		},
		{
			name:        "invalid format - multiple slashes",
			fullName:    "owner/repo/extra",
			expectError: true,
		},
		{
			name:        "invalid format - empty",
			fullName:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := splitRepoFullName(tt.fullName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectOwner, owner)
				assert.Equal(t, tt.expectRepo, repo)
			}
		})
	}
}

// Note: Tests for Read() method that require GitHub API calls should be
// implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
