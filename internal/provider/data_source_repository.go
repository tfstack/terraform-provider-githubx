package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &repositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &repositoryDataSource{}
)

// NewRepositoryDataSource is a helper function to simplify the provider implementation.
func NewRepositoryDataSource() datasource.DataSource {
	return &repositoryDataSource{}
}

// repositoryDataSource is the data source implementation.
type repositoryDataSource struct {
	client *github.Client
	owner  string
}

// repositoryDataSourceModel maps the data source schema data.
type repositoryDataSourceModel struct {
	FullName                 types.String `tfsdk:"full_name"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	HomepageURL              types.String `tfsdk:"homepage_url"`
	Private                  types.Bool   `tfsdk:"private"`
	Visibility               types.String `tfsdk:"visibility"`
	HasIssues                types.Bool   `tfsdk:"has_issues"`
	HasDiscussions           types.Bool   `tfsdk:"has_discussions"`
	HasProjects              types.Bool   `tfsdk:"has_projects"`
	HasDownloads             types.Bool   `tfsdk:"has_downloads"`
	HasWiki                  types.Bool   `tfsdk:"has_wiki"`
	IsTemplate               types.Bool   `tfsdk:"is_template"`
	Fork                     types.Bool   `tfsdk:"fork"`
	AllowMergeCommit         types.Bool   `tfsdk:"allow_merge_commit"`
	AllowSquashMerge         types.Bool   `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge         types.Bool   `tfsdk:"allow_rebase_merge"`
	AllowAutoMerge           types.Bool   `tfsdk:"allow_auto_merge"`
	AllowUpdateBranch        types.Bool   `tfsdk:"allow_update_branch"`
	SquashMergeCommitTitle   types.String `tfsdk:"squash_merge_commit_title"`
	SquashMergeCommitMessage types.String `tfsdk:"squash_merge_commit_message"`
	MergeCommitTitle         types.String `tfsdk:"merge_commit_title"`
	MergeCommitMessage       types.String `tfsdk:"merge_commit_message"`
	DefaultBranch            types.String `tfsdk:"default_branch"`
	PrimaryLanguage          types.String `tfsdk:"primary_language"`
	Archived                 types.Bool   `tfsdk:"archived"`
	RepositoryLicense        types.Object `tfsdk:"repository_license"`
	Pages                    types.Object `tfsdk:"pages"`
	Topics                   types.List   `tfsdk:"topics"`
	HTMLURL                  types.String `tfsdk:"html_url"`
	SSHCloneURL              types.String `tfsdk:"ssh_clone_url"`
	SVNURL                   types.String `tfsdk:"svn_url"`
	GitCloneURL              types.String `tfsdk:"git_clone_url"`
	HTTPCloneURL             types.String `tfsdk:"http_clone_url"`
	Template                 types.Object `tfsdk:"template"`
	NodeID                   types.String `tfsdk:"node_id"`
	RepoID                   types.Int64  `tfsdk:"repo_id"`
	DeleteBranchOnMerge      types.Bool   `tfsdk:"delete_branch_on_merge"`
	ID                       types.String `tfsdk:"id"`
}

// pagesModel represents GitHub Pages configuration.
type pagesModel struct {
	Source    types.Object `tfsdk:"source"`
	BuildType types.String `tfsdk:"build_type"`
	CNAME     types.String `tfsdk:"cname"`
	Custom404 types.Bool   `tfsdk:"custom_404"`
	HTMLURL   types.String `tfsdk:"html_url"`
	Status    types.String `tfsdk:"status"`
	URL       types.String `tfsdk:"url"`
}

// pagesSourceModel represents the Pages source configuration.
type pagesSourceModel struct {
	Branch types.String `tfsdk:"branch"`
	Path   types.String `tfsdk:"path"`
}

// Metadata returns the data source type name.
func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

// Schema defines the schema for the data source.
func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub repository.",
		Attributes: map[string]schema.Attribute{
			"full_name": schema.StringAttribute{
				Description: "The full name of the repository (owner/repo). Conflicts with `name`.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository. Conflicts with `full_name`. If `name` is provided, the provider-level `owner` configuration will be used.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the repository.",
				Computed:    true,
			},
			"homepage_url": schema.StringAttribute{
				Description: "The homepage URL of the repository.",
				Computed:    true,
			},
			"private": schema.BoolAttribute{
				Description: "Whether the repository is private.",
				Computed:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "The visibility of the repository (public, private, or internal).",
				Computed:    true,
			},
			"has_issues": schema.BoolAttribute{
				Description: "Whether the repository has issues enabled.",
				Computed:    true,
			},
			"has_discussions": schema.BoolAttribute{
				Description: "Whether the repository has discussions enabled.",
				Computed:    true,
			},
			"has_projects": schema.BoolAttribute{
				Description: "Whether the repository has projects enabled.",
				Computed:    true,
			},
			"has_downloads": schema.BoolAttribute{
				Description: "Whether the repository has downloads enabled.",
				Computed:    true,
			},
			"has_wiki": schema.BoolAttribute{
				Description: "Whether the repository has wiki enabled.",
				Computed:    true,
			},
			"is_template": schema.BoolAttribute{
				Description: "Whether the repository is a template.",
				Computed:    true,
			},
			"fork": schema.BoolAttribute{
				Description: "Whether the repository is a fork.",
				Computed:    true,
			},
			"allow_merge_commit": schema.BoolAttribute{
				Description: "Whether merge commits are allowed.",
				Computed:    true,
			},
			"allow_squash_merge": schema.BoolAttribute{
				Description: "Whether squash merges are allowed.",
				Computed:    true,
			},
			"allow_rebase_merge": schema.BoolAttribute{
				Description: "Whether rebase merges are allowed.",
				Computed:    true,
			},
			"allow_auto_merge": schema.BoolAttribute{
				Description: "Whether auto-merge is enabled.",
				Computed:    true,
			},
			"allow_update_branch": schema.BoolAttribute{
				Description: "Whether branch updates are allowed.",
				Computed:    true,
			},
			"squash_merge_commit_title": schema.StringAttribute{
				Description: "The default commit title for squash merges.",
				Computed:    true,
			},
			"squash_merge_commit_message": schema.StringAttribute{
				Description: "The default commit message for squash merges.",
				Computed:    true,
			},
			"merge_commit_title": schema.StringAttribute{
				Description: "The default commit title for merge commits.",
				Computed:    true,
			},
			"merge_commit_message": schema.StringAttribute{
				Description: "The default commit message for merge commits.",
				Computed:    true,
			},
			"default_branch": schema.StringAttribute{
				Description: "The default branch of the repository.",
				Computed:    true,
			},
			"primary_language": schema.StringAttribute{
				Description: "The primary programming language of the repository.",
				Computed:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the repository is archived.",
				Computed:    true,
			},
			"repository_license": schema.SingleNestedAttribute{
				Description: "The license information for the repository.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed: true,
					},
					"path": schema.StringAttribute{
						Computed: true,
					},
					"license": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"key":            schema.StringAttribute{Computed: true},
							"name":           schema.StringAttribute{Computed: true},
							"url":            schema.StringAttribute{Computed: true},
							"spdx_id":        schema.StringAttribute{Computed: true},
							"html_url":       schema.StringAttribute{Computed: true},
							"featured":       schema.BoolAttribute{Computed: true},
							"description":    schema.StringAttribute{Computed: true},
							"implementation": schema.StringAttribute{Computed: true},
							"permissions":    schema.ListAttribute{ElementType: types.StringType, Computed: true},
							"conditions":     schema.ListAttribute{ElementType: types.StringType, Computed: true},
							"limitations":    schema.ListAttribute{ElementType: types.StringType, Computed: true},
							"body":           schema.StringAttribute{Computed: true},
						},
					},
					"sha":          schema.StringAttribute{Computed: true},
					"size":         schema.Int64Attribute{Computed: true},
					"url":          schema.StringAttribute{Computed: true},
					"html_url":     schema.StringAttribute{Computed: true},
					"git_url":      schema.StringAttribute{Computed: true},
					"download_url": schema.StringAttribute{Computed: true},
					"type":         schema.StringAttribute{Computed: true},
					"content":      schema.StringAttribute{Computed: true},
					"encoding":     schema.StringAttribute{Computed: true},
				},
			},
			"pages": schema.SingleNestedAttribute{
				Description: "The GitHub Pages configuration for the repository.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"source": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"branch": schema.StringAttribute{Computed: true},
							"path":   schema.StringAttribute{Computed: true},
						},
					},
					"build_type": schema.StringAttribute{Computed: true},
					"cname":      schema.StringAttribute{Computed: true},
					"custom_404": schema.BoolAttribute{Computed: true},
					"html_url":   schema.StringAttribute{Computed: true},
					"status":     schema.StringAttribute{Computed: true},
					"url":        schema.StringAttribute{Computed: true},
				},
			},
			"topics": schema.ListAttribute{
				Description: "The topics (tags) associated with the repository.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"html_url": schema.StringAttribute{
				Description: "The HTML URL of the repository.",
				Computed:    true,
			},
			"ssh_clone_url": schema.StringAttribute{
				Description: "The SSH clone URL of the repository.",
				Computed:    true,
			},
			"svn_url": schema.StringAttribute{
				Description: "The SVN URL of the repository.",
				Computed:    true,
			},
			"git_clone_url": schema.StringAttribute{
				Description: "The Git clone URL of the repository.",
				Computed:    true,
			},
			"http_clone_url": schema.StringAttribute{
				Description: "The HTTP clone URL of the repository.",
				Computed:    true,
			},
			"template": schema.SingleNestedAttribute{
				Description: "The template repository information, if this repository was created from a template.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"owner":      schema.StringAttribute{Computed: true},
					"repository": schema.StringAttribute{Computed: true},
				},
			},
			"node_id": schema.StringAttribute{
				Description: "The GitHub node ID of the repository.",
				Computed:    true,
			},
			"repo_id": schema.Int64Attribute{
				Description: "The GitHub repository ID as an integer.",
				Computed:    true,
			},
			"delete_branch_on_merge": schema.BoolAttribute{
				Description: "Whether to delete branches after merging pull requests.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository name).",
				Computed:    true,
			},
		},
	}
}

// Configure enables provider-level data or clients to be set in the
// provider-defined data source type. It is separately executed for each
// ReadDataSource RPC.
func (d *repositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientData, ok := req.ProviderData.(githubxClientData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected githubxClientData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = clientData.Client
	d.owner = clientData.Owner
}

// getOwner gets the owner, falling back to authenticated user if not set.
func (d *repositoryDataSource) getOwner(ctx context.Context) (string, error) {
	if d.owner != "" {
		return d.owner, nil
	}
	// Try to get authenticated user
	user, _, err := d.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and unable to fetch authenticated user: %v", err)
	}
	if user == nil || user.Login == nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and authenticated user information is unavailable")
	}
	return user.GetLogin(), nil
}

// Read refreshes the Terraform state with the latest data.
func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Error",
			"GitHub client is not configured. Please ensure a token is provided.",
		)
		return
	}

	// Determine owner and repo name
	var owner, repoName string
	fullName := data.FullName.ValueString()
	name := data.Name.ValueString()

	// Check for conflicts
	if fullName != "" && name != "" {
		resp.Diagnostics.AddError(
			"Conflicting Attributes",
			"Cannot specify both `full_name` and `name`. Please use only one.",
		)
		return
	}

	// Parse full_name or use name with owner
	if fullName != "" {
		var err error
		owner, repoName, err = splitRepoFullName(fullName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Full Name",
				fmt.Sprintf("Unable to parse full_name: %v", err),
			)
			return
		}
	} else if name != "" {
		repoName = name
		var err error
		owner, err = d.getOwner(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Missing Owner",
				fmt.Sprintf("Either `full_name` must be provided, or `name` must be provided along with provider-level `owner` configuration or authentication. Error: %v", err),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either `full_name` or `name` must be provided.",
		)
		return
	}

	// Fetch the repository from GitHub
	repo, ghResp, err := d.client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			// Check both ghResp and ghErr.Response for 404 status
			if (ghResp != nil && ghResp.StatusCode == http.StatusNotFound) ||
				(ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound) {
				resp.Diagnostics.AddWarning(
					"Repository Not Found",
					fmt.Sprintf("Repository %s/%s not found. Setting empty state.", owner, repoName),
				)
				data.ID = types.StringValue("")
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error fetching GitHub repository",
			fmt.Sprintf("Unable to fetch repository %s/%s: %v", owner, repoName, err),
		)
		return
	}

	// Map basic fields
	data.ID = types.StringValue(repo.GetName())
	data.Name = types.StringValue(repo.GetName())
	data.FullName = types.StringValue(repo.GetFullName())
	data.Description = types.StringValue(repo.GetDescription())
	data.HomepageURL = types.StringValue(repo.GetHomepage())
	data.Private = types.BoolValue(repo.GetPrivate())
	data.Visibility = types.StringValue(repo.GetVisibility())
	data.HasIssues = types.BoolValue(repo.GetHasIssues())
	data.HasDiscussions = types.BoolValue(repo.GetHasDiscussions())
	data.HasProjects = types.BoolValue(repo.GetHasProjects())
	data.HasDownloads = types.BoolValue(repo.GetHasDownloads())
	data.HasWiki = types.BoolValue(repo.GetHasWiki())
	data.IsTemplate = types.BoolValue(repo.GetIsTemplate())
	data.Fork = types.BoolValue(repo.GetFork())
	data.AllowMergeCommit = types.BoolValue(repo.GetAllowMergeCommit())
	data.AllowSquashMerge = types.BoolValue(repo.GetAllowSquashMerge())
	data.AllowRebaseMerge = types.BoolValue(repo.GetAllowRebaseMerge())
	data.AllowAutoMerge = types.BoolValue(repo.GetAllowAutoMerge())
	data.AllowUpdateBranch = types.BoolValue(repo.GetAllowUpdateBranch())
	data.SquashMergeCommitTitle = types.StringValue(repo.GetSquashMergeCommitTitle())
	data.SquashMergeCommitMessage = types.StringValue(repo.GetSquashMergeCommitMessage())
	data.MergeCommitTitle = types.StringValue(repo.GetMergeCommitTitle())
	data.MergeCommitMessage = types.StringValue(repo.GetMergeCommitMessage())
	data.DefaultBranch = types.StringValue(repo.GetDefaultBranch())
	data.PrimaryLanguage = types.StringValue(repo.GetLanguage())
	data.Archived = types.BoolValue(repo.GetArchived())
	data.HTMLURL = types.StringValue(repo.GetHTMLURL())
	data.SSHCloneURL = types.StringValue(repo.GetSSHURL())
	data.SVNURL = types.StringValue(repo.GetSVNURL())
	data.GitCloneURL = types.StringValue(repo.GetGitURL())
	data.HTTPCloneURL = types.StringValue(repo.GetCloneURL())
	data.NodeID = types.StringValue(repo.GetNodeID())
	data.RepoID = types.Int64Value(repo.GetID())
	data.DeleteBranchOnMerge = types.BoolValue(repo.GetDeleteBranchOnMerge())

	// Handle topics
	if repo.Topics != nil {
		topics := make([]types.String, len(repo.Topics))
		for i, topic := range repo.Topics {
			topics[i] = types.StringValue(topic)
		}
		topicsList, diags := types.ListValueFrom(ctx, types.StringType, topics)
		resp.Diagnostics.Append(diags...)
		data.Topics = topicsList
	} else {
		data.Topics = types.ListNull(types.StringType)
	}

	// Handle pages
	if repo.GetHasPages() {
		pages, _, err := d.client.Repositories.GetPagesInfo(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error fetching Pages info",
				fmt.Sprintf("Unable to fetch Pages info: %v", err),
			)
			data.Pages = types.ObjectNull(pagesObjectAttributeTypes())
		} else {
			pagesObj, diags := flattenPages(ctx, pages)
			resp.Diagnostics.Append(diags...)
			data.Pages = pagesObj
		}
	} else {
		data.Pages = types.ObjectNull(pagesObjectAttributeTypes())
	}

	// Handle license
	if repo.License != nil {
		license, _, err := d.client.Repositories.License(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error fetching license",
				fmt.Sprintf("Unable to fetch license: %v", err),
			)
			data.RepositoryLicense = types.ObjectNull(repositoryLicenseObjectAttributeTypes())
		} else {
			licenseObj, diags := flattenRepositoryLicense(ctx, license)
			resp.Diagnostics.Append(diags...)
			data.RepositoryLicense = licenseObj
		}
	} else {
		data.RepositoryLicense = types.ObjectNull(repositoryLicenseObjectAttributeTypes())
	}

	// Handle template
	if repo.TemplateRepository != nil {
		templateObj, diags := flattenTemplate(ctx, repo.TemplateRepository)
		resp.Diagnostics.Append(diags...)
		data.Template = templateObj
	} else {
		data.Template = types.ObjectNull(templateObjectAttributeTypes())
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// splitRepoFullName splits a full repository name (owner/repo) into owner and repo name.
func splitRepoFullName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected full name format (%q), expected owner/repo_name", fullName)
	}
	return parts[0], parts[1], nil
}

// Helper functions for flattening nested objects.
func pagesObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"source":     types.ObjectType{AttrTypes: pagesSourceObjectAttributeTypes()},
		"build_type": types.StringType,
		"cname":      types.StringType,
		"custom_404": types.BoolType,
		"html_url":   types.StringType,
		"status":     types.StringType,
		"url":        types.StringType,
	}
}

func pagesSourceObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"branch": types.StringType,
		"path":   types.StringType,
	}
}

func repositoryLicenseObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":         types.StringType,
		"path":         types.StringType,
		"license":      types.ObjectType{AttrTypes: licenseInfoObjectAttributeTypes()},
		"sha":          types.StringType,
		"size":         types.Int64Type,
		"url":          types.StringType,
		"html_url":     types.StringType,
		"git_url":      types.StringType,
		"download_url": types.StringType,
		"type":         types.StringType,
		"content":      types.StringType,
		"encoding":     types.StringType,
	}
}

func licenseInfoObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":            types.StringType,
		"name":           types.StringType,
		"url":            types.StringType,
		"spdx_id":        types.StringType,
		"html_url":       types.StringType,
		"featured":       types.BoolType,
		"description":    types.StringType,
		"implementation": types.StringType,
		"permissions":    types.ListType{ElemType: types.StringType},
		"conditions":     types.ListType{ElemType: types.StringType},
		"limitations":    types.ListType{ElemType: types.StringType},
		"body":           types.StringType,
	}
}

func templateObjectAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"owner":      types.StringType,
		"repository": types.StringType,
	}
}

func flattenPages(_ context.Context, pages *github.Pages) (types.Object, diag.Diagnostics) {
	if pages == nil {
		return types.ObjectNull(pagesObjectAttributeTypes()), nil
	}

	var sourceObj types.Object
	if pages.Source != nil {
		sourceAttrs := map[string]attr.Value{
			"branch": types.StringValue(pages.Source.GetBranch()),
			"path":   types.StringValue(pages.Source.GetPath()),
		}
		var diags diag.Diagnostics
		sourceObj, diags = types.ObjectValue(pagesSourceObjectAttributeTypes(), sourceAttrs)
		if diags.HasError() {
			return types.ObjectNull(pagesObjectAttributeTypes()), diags
		}
	} else {
		sourceObj = types.ObjectNull(pagesSourceObjectAttributeTypes())
	}

	attrs := map[string]attr.Value{
		"source":     sourceObj,
		"build_type": types.StringValue(pages.GetBuildType()),
		"cname":      types.StringValue(pages.GetCNAME()),
		"custom_404": types.BoolValue(pages.GetCustom404()),
		"html_url":   types.StringValue(pages.GetHTMLURL()),
		"status":     types.StringValue(pages.GetStatus()),
		"url":        types.StringValue(pages.GetURL()),
	}

	return types.ObjectValue(pagesObjectAttributeTypes(), attrs)
}

func flattenRepositoryLicense(ctx context.Context, license *github.RepositoryLicense) (types.Object, diag.Diagnostics) {
	if license == nil {
		return types.ObjectNull(repositoryLicenseObjectAttributeTypes()), nil
	}

	var licenseInfoObj types.Object
	if license.License != nil {
		licenseInfoAttrs := map[string]attr.Value{
			"key":            types.StringValue(license.License.GetKey()),
			"name":           types.StringValue(license.License.GetName()),
			"url":            types.StringValue(license.License.GetURL()),
			"spdx_id":        types.StringValue(license.License.GetSPDXID()),
			"html_url":       types.StringValue(license.License.GetHTMLURL()),
			"featured":       types.BoolValue(license.License.GetFeatured()),
			"description":    types.StringValue(license.License.GetDescription()),
			"implementation": types.StringValue(license.License.GetImplementation()),
			"body":           types.StringValue(license.License.GetBody()),
		}

		// Handle permissions, conditions, limitations
		var permissionsList types.List
		permissions := license.License.GetPermissions()
		if len(permissions) > 0 {
			perms := make([]types.String, len(permissions))
			for i, p := range permissions {
				perms[i] = types.StringValue(p)
			}
			var diags diag.Diagnostics
			permissionsList, diags = types.ListValueFrom(ctx, types.StringType, perms)
			if diags.HasError() {
				return types.ObjectNull(repositoryLicenseObjectAttributeTypes()), diags
			}
		} else {
			permissionsList = types.ListNull(types.StringType)
		}

		var conditionsList types.List
		conditions := license.License.GetConditions()
		if len(conditions) > 0 {
			conds := make([]types.String, len(conditions))
			for i, c := range conditions {
				conds[i] = types.StringValue(c)
			}
			var diags diag.Diagnostics
			conditionsList, diags = types.ListValueFrom(ctx, types.StringType, conds)
			if diags.HasError() {
				return types.ObjectNull(repositoryLicenseObjectAttributeTypes()), diags
			}
		} else {
			conditionsList = types.ListNull(types.StringType)
		}

		var limitationsList types.List
		limitations := license.License.GetLimitations()
		if len(limitations) > 0 {
			lims := make([]types.String, len(limitations))
			for i, l := range limitations {
				lims[i] = types.StringValue(l)
			}
			var diags diag.Diagnostics
			limitationsList, diags = types.ListValueFrom(ctx, types.StringType, lims)
			if diags.HasError() {
				return types.ObjectNull(repositoryLicenseObjectAttributeTypes()), diags
			}
		} else {
			limitationsList = types.ListNull(types.StringType)
		}

		licenseInfoAttrs["permissions"] = permissionsList
		licenseInfoAttrs["conditions"] = conditionsList
		licenseInfoAttrs["limitations"] = limitationsList

		var diags diag.Diagnostics
		licenseInfoObj, diags = types.ObjectValue(licenseInfoObjectAttributeTypes(), licenseInfoAttrs)
		if diags.HasError() {
			return types.ObjectNull(repositoryLicenseObjectAttributeTypes()), diags
		}
	} else {
		licenseInfoObj = types.ObjectNull(licenseInfoObjectAttributeTypes())
	}

	attrs := map[string]attr.Value{
		"name":         types.StringValue(license.GetName()),
		"path":         types.StringValue(license.GetPath()),
		"license":      licenseInfoObj,
		"sha":          types.StringValue(license.GetSHA()),
		"size":         types.Int64Value(int64(license.GetSize())),
		"url":          types.StringValue(license.GetURL()),
		"html_url":     types.StringValue(license.GetHTMLURL()),
		"git_url":      types.StringValue(license.GetGitURL()),
		"download_url": types.StringValue(license.GetDownloadURL()),
		"type":         types.StringValue(license.GetType()),
		"content":      types.StringValue(license.GetContent()),
		"encoding":     types.StringValue(license.GetEncoding()),
	}

	return types.ObjectValue(repositoryLicenseObjectAttributeTypes(), attrs)
}

func flattenTemplate(_ context.Context, template *github.Repository) (types.Object, diag.Diagnostics) {
	if template == nil {
		return types.ObjectNull(templateObjectAttributeTypes()), nil
	}

	attrs := map[string]attr.Value{
		"owner":      types.StringValue(template.Owner.GetLogin()),
		"repository": types.StringValue(template.GetName()),
	}

	return types.ObjectValue(templateObjectAttributeTypes(), attrs)
}
