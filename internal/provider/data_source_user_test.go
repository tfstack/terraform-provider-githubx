package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestUserDataSource_Metadata(t *testing.T) {
	ds := NewUserDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_user", resp.TypeName)
}

func TestUserDataSource_Schema(t *testing.T) {
	ds := NewUserDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Get information on a GitHub user")

	// Check required attribute
	usernameAttr, ok := resp.Schema.Attributes["username"]
	assert.True(t, ok)
	assert.True(t, usernameAttr.IsRequired())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	userIDAttr, ok := resp.Schema.Attributes["user_id"]
	assert.True(t, ok)
	assert.True(t, userIDAttr.IsComputed())

	nodeIDAttr, ok := resp.Schema.Attributes["node_id"]
	assert.True(t, ok)
	assert.True(t, nodeIDAttr.IsComputed())

	avatarURLAttr, ok := resp.Schema.Attributes["avatar_url"]
	assert.True(t, ok)
	assert.True(t, avatarURLAttr.IsComputed())

	htmlURLAttr, ok := resp.Schema.Attributes["html_url"]
	assert.True(t, ok)
	assert.True(t, htmlURLAttr.IsComputed())

	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsComputed())

	companyAttr, ok := resp.Schema.Attributes["company"]
	assert.True(t, ok)
	assert.True(t, companyAttr.IsComputed())

	blogAttr, ok := resp.Schema.Attributes["blog"]
	assert.True(t, ok)
	assert.True(t, blogAttr.IsComputed())

	locationAttr, ok := resp.Schema.Attributes["location"]
	assert.True(t, ok)
	assert.True(t, locationAttr.IsComputed())

	emailAttr, ok := resp.Schema.Attributes["email"]
	assert.True(t, ok)
	assert.True(t, emailAttr.IsComputed())

	bioAttr, ok := resp.Schema.Attributes["bio"]
	assert.True(t, ok)
	assert.True(t, bioAttr.IsComputed())

	publicReposAttr, ok := resp.Schema.Attributes["public_repos"]
	assert.True(t, ok)
	assert.True(t, publicReposAttr.IsComputed())

	publicGistsAttr, ok := resp.Schema.Attributes["public_gists"]
	assert.True(t, ok)
	assert.True(t, publicGistsAttr.IsComputed())

	followersAttr, ok := resp.Schema.Attributes["followers"]
	assert.True(t, ok)
	assert.True(t, followersAttr.IsComputed())

	followingAttr, ok := resp.Schema.Attributes["following"]
	assert.True(t, ok)
	assert.True(t, followingAttr.IsComputed())

	createdAtAttr, ok := resp.Schema.Attributes["created_at"]
	assert.True(t, ok)
	assert.True(t, createdAtAttr.IsComputed())

	updatedAtAttr, ok := resp.Schema.Attributes["updated_at"]
	assert.True(t, ok)
	assert.True(t, updatedAtAttr.IsComputed())
}

func TestUserDataSource_Configure(t *testing.T) {
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
			ds := &userDataSource{}
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
				// If provider data is valid, verify client is set
				if tt.providerData != nil {
					clientData, ok := tt.providerData.(githubxClientData)
					if ok {
						assert.Equal(t, clientData.Client, ds.client)
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
