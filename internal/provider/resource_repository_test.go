package provider

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryResource_Metadata(t *testing.T) {
	r := NewRepositoryResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "githubx",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(t.Context(), req, resp)

	assert.Equal(t, "githubx_repository", resp.TypeName)
}

func TestRepositoryResource_Schema(t *testing.T) {
	r := NewRepositoryResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a GitHub repository")

	// Check required attribute
	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsRequired())

	// Check optional attributes
	descriptionAttr, ok := resp.Schema.Attributes["description"]
	assert.True(t, ok)
	assert.True(t, descriptionAttr.IsOptional())

	homepageURLAttr, ok := resp.Schema.Attributes["homepage_url"]
	assert.True(t, ok)
	assert.True(t, homepageURLAttr.IsOptional())

	visibilityAttr, ok := resp.Schema.Attributes["visibility"]
	assert.True(t, ok)
	assert.True(t, visibilityAttr.IsOptional())
	assert.True(t, visibilityAttr.IsComputed())

	hasIssuesAttr, ok := resp.Schema.Attributes["has_issues"]
	assert.True(t, ok)
	assert.True(t, hasIssuesAttr.IsOptional())
	assert.True(t, hasIssuesAttr.IsComputed())

	hasDiscussionsAttr, ok := resp.Schema.Attributes["has_discussions"]
	assert.True(t, ok)
	assert.True(t, hasDiscussionsAttr.IsOptional())
	assert.True(t, hasDiscussionsAttr.IsComputed())

	hasProjectsAttr, ok := resp.Schema.Attributes["has_projects"]
	assert.True(t, ok)
	assert.True(t, hasProjectsAttr.IsOptional())
	assert.True(t, hasProjectsAttr.IsComputed())

	hasDownloadsAttr, ok := resp.Schema.Attributes["has_downloads"]
	assert.True(t, ok)
	assert.True(t, hasDownloadsAttr.IsOptional())
	assert.True(t, hasDownloadsAttr.IsComputed())

	hasWikiAttr, ok := resp.Schema.Attributes["has_wiki"]
	assert.True(t, ok)
	assert.True(t, hasWikiAttr.IsOptional())
	assert.True(t, hasWikiAttr.IsComputed())

	isTemplateAttr, ok := resp.Schema.Attributes["is_template"]
	assert.True(t, ok)
	assert.True(t, isTemplateAttr.IsOptional())
	assert.True(t, isTemplateAttr.IsComputed())

	allowMergeCommitAttr, ok := resp.Schema.Attributes["allow_merge_commit"]
	assert.True(t, ok)
	assert.True(t, allowMergeCommitAttr.IsOptional())
	assert.True(t, allowMergeCommitAttr.IsComputed())

	allowSquashMergeAttr, ok := resp.Schema.Attributes["allow_squash_merge"]
	assert.True(t, ok)
	assert.True(t, allowSquashMergeAttr.IsOptional())
	assert.True(t, allowSquashMergeAttr.IsComputed())

	allowRebaseMergeAttr, ok := resp.Schema.Attributes["allow_rebase_merge"]
	assert.True(t, ok)
	assert.True(t, allowRebaseMergeAttr.IsOptional())
	assert.True(t, allowRebaseMergeAttr.IsComputed())

	allowAutoMergeAttr, ok := resp.Schema.Attributes["allow_auto_merge"]
	assert.True(t, ok)
	assert.True(t, allowAutoMergeAttr.IsOptional())
	assert.True(t, allowAutoMergeAttr.IsComputed())

	allowUpdateBranchAttr, ok := resp.Schema.Attributes["allow_update_branch"]
	assert.True(t, ok)
	assert.True(t, allowUpdateBranchAttr.IsOptional())
	assert.True(t, allowUpdateBranchAttr.IsComputed())

	squashMergeCommitTitleAttr, ok := resp.Schema.Attributes["squash_merge_commit_title"]
	assert.True(t, ok)
	assert.True(t, squashMergeCommitTitleAttr.IsOptional())
	assert.True(t, squashMergeCommitTitleAttr.IsComputed())

	squashMergeCommitMessageAttr, ok := resp.Schema.Attributes["squash_merge_commit_message"]
	assert.True(t, ok)
	assert.True(t, squashMergeCommitMessageAttr.IsOptional())
	assert.True(t, squashMergeCommitMessageAttr.IsComputed())

	mergeCommitTitleAttr, ok := resp.Schema.Attributes["merge_commit_title"]
	assert.True(t, ok)
	assert.True(t, mergeCommitTitleAttr.IsOptional())
	assert.True(t, mergeCommitTitleAttr.IsComputed())

	mergeCommitMessageAttr, ok := resp.Schema.Attributes["merge_commit_message"]
	assert.True(t, ok)
	assert.True(t, mergeCommitMessageAttr.IsOptional())
	assert.True(t, mergeCommitMessageAttr.IsComputed())

	deleteBranchOnMergeAttr, ok := resp.Schema.Attributes["delete_branch_on_merge"]
	assert.True(t, ok)
	assert.True(t, deleteBranchOnMergeAttr.IsOptional())
	assert.True(t, deleteBranchOnMergeAttr.IsComputed())

	archiveOnDestroyAttr, ok := resp.Schema.Attributes["archive_on_destroy"]
	assert.True(t, ok)
	assert.True(t, archiveOnDestroyAttr.IsOptional())

	autoInitAttr, ok := resp.Schema.Attributes["auto_init"]
	assert.True(t, ok)
	assert.True(t, autoInitAttr.IsOptional())
	assert.True(t, autoInitAttr.IsComputed())

	topicsAttr, ok := resp.Schema.Attributes["topics"]
	assert.True(t, ok)
	assert.True(t, topicsAttr.IsOptional())

	vulnerabilityAlertsAttr, ok := resp.Schema.Attributes["vulnerability_alerts"]
	assert.True(t, ok)
	assert.True(t, vulnerabilityAlertsAttr.IsOptional())
	assert.True(t, vulnerabilityAlertsAttr.IsComputed())

	pagesAttr, ok := resp.Schema.Attributes["pages"]
	assert.True(t, ok)
	assert.True(t, pagesAttr.IsOptional())
	assert.True(t, pagesAttr.IsComputed())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())

	fullNameAttr, ok := resp.Schema.Attributes["full_name"]
	assert.True(t, ok)
	assert.True(t, fullNameAttr.IsComputed())

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

	archivedAttr, ok := resp.Schema.Attributes["archived"]
	assert.True(t, ok)
	assert.True(t, archivedAttr.IsComputed())
}

func TestRepositoryResource_Configure(t *testing.T) {
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
			rs := &repositoryResource{}
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
