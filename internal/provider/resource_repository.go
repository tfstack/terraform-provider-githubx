package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &repositoryResource{}
	_ resource.ResourceWithConfigure   = &repositoryResource{}
	_ resource.ResourceWithImportState = &repositoryResource{}
)

// NewRepositoryResource is a helper function to simplify the provider implementation.
func NewRepositoryResource() resource.Resource {
	return &repositoryResource{}
}

// repositoryResource is the resource implementation.
type repositoryResource struct {
	client            *github.Client
	owner             string
	authenticatedUser string // Store authenticated user login for comparison
}

// repositoryResourceModel maps the resource schema data.
type repositoryResourceModel struct {
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	HomepageURL              types.String `tfsdk:"homepage_url"`
	Visibility               types.String `tfsdk:"visibility"`
	HasIssues                types.Bool   `tfsdk:"has_issues"`
	HasDiscussions           types.Bool   `tfsdk:"has_discussions"`
	HasProjects              types.Bool   `tfsdk:"has_projects"`
	HasDownloads             types.Bool   `tfsdk:"has_downloads"`
	HasWiki                  types.Bool   `tfsdk:"has_wiki"`
	IsTemplate               types.Bool   `tfsdk:"is_template"`
	AllowMergeCommit         types.Bool   `tfsdk:"allow_merge_commit"`
	AllowSquashMerge         types.Bool   `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge         types.Bool   `tfsdk:"allow_rebase_merge"`
	AllowAutoMerge           types.Bool   `tfsdk:"allow_auto_merge"`
	AllowUpdateBranch        types.Bool   `tfsdk:"allow_update_branch"`
	SquashMergeCommitTitle   types.String `tfsdk:"squash_merge_commit_title"`
	SquashMergeCommitMessage types.String `tfsdk:"squash_merge_commit_message"`
	MergeCommitTitle         types.String `tfsdk:"merge_commit_title"`
	MergeCommitMessage       types.String `tfsdk:"merge_commit_message"`
	DeleteBranchOnMerge      types.Bool   `tfsdk:"delete_branch_on_merge"`
	ArchiveOnDestroy         types.Bool   `tfsdk:"archive_on_destroy"`
	Archived                 types.Bool   `tfsdk:"archived"`
	AutoInit                 types.Bool   `tfsdk:"auto_init"`
	Pages                    types.Object `tfsdk:"pages"`
	Topics                   types.Set    `tfsdk:"topics"`
	VulnerabilityAlerts      types.Bool   `tfsdk:"vulnerability_alerts"`
	ID                       types.String `tfsdk:"id"`
	FullName                 types.String `tfsdk:"full_name"`
	DefaultBranch            types.String `tfsdk:"default_branch"`
	HTMLURL                  types.String `tfsdk:"html_url"`
	NodeID                   types.String `tfsdk:"node_id"`
	RepoID                   types.Int64  `tfsdk:"repo_id"`
}

// Metadata returns the resource type name.
func (r *repositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

// Schema defines the schema for the resource.
func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a GitHub repository.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[-a-zA-Z0-9_.]{1,100}$`),
						"must include only alphanumeric characters, underscores or hyphens and consist of 100 characters or less",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the repository.",
				Optional:    true,
			},
			"homepage_url": schema.StringAttribute{
				Description: "URL of a page describing the project.",
				Optional:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "Can be 'public' or 'private'. If your organization is associated with an enterprise account using GitHub Enterprise Cloud or GitHub Enterprise Server 2.20+, visibility can also be 'internal'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private", "internal"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"has_issues": schema.BoolAttribute{
				Description: "Whether the repository has issues enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_discussions": schema.BoolAttribute{
				Description: "Whether the repository has discussions enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_projects": schema.BoolAttribute{
				Description: "Whether the repository has projects enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_downloads": schema.BoolAttribute{
				Description: "Whether the repository has downloads enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_wiki": schema.BoolAttribute{
				Description: "Whether the repository has wiki enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_template": schema.BoolAttribute{
				Description: "Whether the repository is a template.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_merge_commit": schema.BoolAttribute{
				Description: "Whether merge commits are allowed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_squash_merge": schema.BoolAttribute{
				Description: "Whether squash merges are allowed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_rebase_merge": schema.BoolAttribute{
				Description: "Whether rebase merges are allowed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_auto_merge": schema.BoolAttribute{
				Description: "Whether auto-merge is enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_update_branch": schema.BoolAttribute{
				Description: "Whether branch updates are allowed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"squash_merge_commit_title": schema.StringAttribute{
				Description: "The default commit title for squash merges. Can be 'PR_TITLE' or 'COMMIT_OR_PR_TITLE'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PR_TITLE", "COMMIT_OR_PR_TITLE"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"squash_merge_commit_message": schema.StringAttribute{
				Description: "The default commit message for squash merges. Can be 'PR_BODY', 'COMMIT_MESSAGES', or 'BLANK'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PR_BODY", "COMMIT_MESSAGES", "BLANK"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"merge_commit_title": schema.StringAttribute{
				Description: "The default commit title for merge commits. Can be 'PR_TITLE' or 'MERGE_MESSAGE'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PR_TITLE", "MERGE_MESSAGE"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"merge_commit_message": schema.StringAttribute{
				Description: "The default commit message for merge commits. Can be 'PR_BODY', 'PR_TITLE', or 'BLANK'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PR_BODY", "PR_TITLE", "BLANK"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"delete_branch_on_merge": schema.BoolAttribute{
				Description: "Whether to delete branches after merging pull requests.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"archive_on_destroy": schema.BoolAttribute{
				Description: "Whether to archive the repository instead of deleting it when the resource is destroyed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the repository is archived.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_init": schema.BoolAttribute{
				Description: "Whether to initialize the repository with a README file. This will create the default branch.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"pages": schema.SingleNestedAttribute{
				Description: "The GitHub Pages configuration for the repository.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"source": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"branch": schema.StringAttribute{
								Required: true,
							},
							"path": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString("/"),
							},
						},
					},
					"build_type": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.OneOf("legacy", "workflow"),
						},
					},
					"cname": schema.StringAttribute{
						Optional: true,
					},
					"custom_404": schema.BoolAttribute{
						Computed: true,
					},
					"html_url": schema.StringAttribute{
						Computed: true,
					},
					"status": schema.StringAttribute{
						Computed: true,
					},
					"url": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"topics": schema.SetAttribute{
				Description: "The topics (tags) associated with the repository. Order does not matter as topics are stored as a set.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"vulnerability_alerts": schema.BoolAttribute{
				Description: "Whether vulnerability alerts are enabled for the repository.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The repository name (same as `name`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the repository (owner/repo).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_branch": schema.StringAttribute{
				Description: "The default branch of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"html_url": schema.StringAttribute{
				Description: "The HTML URL of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_id": schema.StringAttribute{
				Description: "The GitHub node ID of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repo_id": schema.Int64Attribute{
				Description: "The GitHub repository ID as an integer.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure enables provider-level data or clients to be set in the
// provider-defined resource type.
func (r *repositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientData, ok := req.ProviderData.(githubxClientData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected githubxClientData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = clientData.Client
	r.owner = clientData.Owner

	// Try to get authenticated user for comparison
	if r.client != nil {
		user, _, err := r.client.Users.Get(ctx, "")
		if err == nil && user != nil && user.Login != nil {
			r.authenticatedUser = user.GetLogin()
		}
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Error",
			"GitHub client is not configured. Please ensure a token is provided.",
		)
		return
	}

	// Get owner, falling back to authenticated user if not set
	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	if plan.Topics.IsUnknown() {
		plan.Topics = types.SetNull(types.StringType)
	}

	repoReq := &github.Repository{
		Name:        github.String(plan.Name.ValueString()),
		Description: github.String(plan.Description.ValueString()),
	}

	if !plan.HomepageURL.IsNull() && plan.HomepageURL.ValueString() != "" {
		repoReq.Homepage = github.String(plan.HomepageURL.ValueString())
	}

	if !plan.Visibility.IsNull() {
		visibility := plan.Visibility.ValueString()
		repoReq.Visibility = github.String(visibility)
		if visibility == "private" {
			repoReq.Private = github.Bool(true)
		} else {
			repoReq.Private = github.Bool(false)
		}
	}

	if !plan.HasIssues.IsNull() {
		repoReq.HasIssues = github.Bool(plan.HasIssues.ValueBool())
	}
	if !plan.HasDiscussions.IsNull() {
		repoReq.HasDiscussions = github.Bool(plan.HasDiscussions.ValueBool())
	}
	if !plan.HasProjects.IsNull() {
		repoReq.HasProjects = github.Bool(plan.HasProjects.ValueBool())
	}
	if !plan.HasDownloads.IsNull() {
		repoReq.HasDownloads = github.Bool(plan.HasDownloads.ValueBool())
	}
	if !plan.HasWiki.IsNull() {
		repoReq.HasWiki = github.Bool(plan.HasWiki.ValueBool())
	}
	if !plan.IsTemplate.IsNull() {
		repoReq.IsTemplate = github.Bool(plan.IsTemplate.ValueBool())
	}

	// GitHub requires at least one merge method to be enabled
	hasMergeCommit := !plan.AllowMergeCommit.IsNull()
	hasSquashMerge := !plan.AllowSquashMerge.IsNull()
	hasRebaseMerge := !plan.AllowRebaseMerge.IsNull()

	var allowMergeCommit, allowSquashMerge, allowRebaseMerge bool

	if !hasMergeCommit && !hasSquashMerge && !hasRebaseMerge {
		allowMergeCommit = true
		allowSquashMerge = true
		allowRebaseMerge = true
	} else {
		if hasMergeCommit {
			allowMergeCommit = plan.AllowMergeCommit.ValueBool()
		} else {
			allowMergeCommit = true
		}

		if hasSquashMerge {
			allowSquashMerge = plan.AllowSquashMerge.ValueBool()
		} else {
			allowSquashMerge = true
		}

		if hasRebaseMerge {
			allowRebaseMerge = plan.AllowRebaseMerge.ValueBool()
		} else {
			allowRebaseMerge = true
		}

		if !allowMergeCommit && !allowSquashMerge && !allowRebaseMerge {
			allowMergeCommit = true
		}
	}

	repoReq.AllowMergeCommit = github.Bool(allowMergeCommit)
	repoReq.AllowSquashMerge = github.Bool(allowSquashMerge)
	repoReq.AllowRebaseMerge = github.Bool(allowRebaseMerge)
	if !plan.AllowAutoMerge.IsNull() {
		repoReq.AllowAutoMerge = github.Bool(plan.AllowAutoMerge.ValueBool())
	}
	if !plan.AllowUpdateBranch.IsNull() {
		repoReq.AllowUpdateBranch = github.Bool(plan.AllowUpdateBranch.ValueBool())
	}

	// Only set squash merge commit settings if squash merge is enabled
	// GitHub requires squash merge to be enabled to set these fields
	if allowSquashMerge {
		if !plan.SquashMergeCommitTitle.IsNull() && !plan.SquashMergeCommitTitle.IsUnknown() && plan.SquashMergeCommitTitle.ValueString() != "" {
			repoReq.SquashMergeCommitTitle = github.String(plan.SquashMergeCommitTitle.ValueString())
		}
		if !plan.SquashMergeCommitMessage.IsNull() && !plan.SquashMergeCommitMessage.IsUnknown() && plan.SquashMergeCommitMessage.ValueString() != "" {
			repoReq.SquashMergeCommitMessage = github.String(plan.SquashMergeCommitMessage.ValueString())
		}
	}

	if allowMergeCommit {
		hasMergeTitle := !plan.MergeCommitTitle.IsNull() && !plan.MergeCommitTitle.IsUnknown() && plan.MergeCommitTitle.ValueString() != ""
		hasMergeMessage := !plan.MergeCommitMessage.IsNull() && !plan.MergeCommitMessage.IsUnknown() && plan.MergeCommitMessage.ValueString() != ""

		if hasMergeTitle && hasMergeMessage {
			repoReq.MergeCommitTitle = github.String(plan.MergeCommitTitle.ValueString())
			repoReq.MergeCommitMessage = github.String(plan.MergeCommitMessage.ValueString())
		}
	}
	if !plan.DeleteBranchOnMerge.IsNull() {
		repoReq.DeleteBranchOnMerge = github.Bool(plan.DeleteBranchOnMerge.ValueBool())
	}

	if !plan.AutoInit.IsNull() {
		repoReq.AutoInit = github.Bool(plan.AutoInit.ValueBool())
	}

	createOwner := owner
	if r.authenticatedUser != "" && owner == r.authenticatedUser {
		createOwner = "" // Empty string creates under authenticated user
	}
	repo, _, err := r.client.Repositories.Create(ctx, createOwner, repoReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository",
			fmt.Sprintf("Unable to create repository %s: %v", plan.Name.ValueString(), err),
		)
		return
	}

	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		topics := make([]string, 0, len(plan.Topics.Elements()))
		resp.Diagnostics.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		if !resp.Diagnostics.HasError() && len(topics) > 0 {
			sort.Strings(topics)
			_, _, err := r.client.Repositories.ReplaceAllTopics(ctx, owner, repo.GetName(), topics)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error setting topics",
					fmt.Sprintf("Unable to set topics: %v", err),
				)
			}
		}
	}

	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		pageDiags := r.updatePages(ctx, owner, repo.GetName(), plan.Pages)
		for _, d := range pageDiags {
			resp.Diagnostics.AddWarning(
				d.Summary(),
				d.Detail(),
			)
		}
	}

	if !plan.VulnerabilityAlerts.IsNull() {
		diags := r.updateVulnerabilityAlerts(ctx, owner, repo.GetName(), plan.VulnerabilityAlerts.ValueBool())
		resp.Diagnostics.Append(diags...)
	}

	explicitHasWiki := plan.HasWiki
	explicitHasIssues := plan.HasIssues
	explicitHasProjects := plan.HasProjects
	explicitHasDownloads := plan.HasDownloads
	explicitHasDiscussions := plan.HasDiscussions
	explicitPages := plan.Pages

	r.readRepository(ctx, owner, repo.GetName(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !explicitHasWiki.IsNull() && !explicitHasWiki.IsUnknown() {
		plan.HasWiki = explicitHasWiki
	}
	if !explicitHasIssues.IsNull() && !explicitHasIssues.IsUnknown() {
		plan.HasIssues = explicitHasIssues
	}
	if !explicitHasProjects.IsNull() && !explicitHasProjects.IsUnknown() {
		plan.HasProjects = explicitHasProjects
	}
	if !explicitHasDownloads.IsNull() && !explicitHasDownloads.IsUnknown() {
		plan.HasDownloads = explicitHasDownloads
	}
	if !explicitHasDiscussions.IsNull() && !explicitHasDiscussions.IsUnknown() {
		plan.HasDiscussions = explicitHasDiscussions
	}

	if !explicitPages.IsNull() && !explicitPages.IsUnknown() {
		pages, _, err := r.client.Repositories.GetPagesInfo(ctx, owner, repo.GetName())
		if err == nil && pages != nil {
			githubPagesObj, pageDiags := flattenPages(ctx, pages)
			if !pageDiags.HasError() {
				mergedPages, mergeDiags := r.mergePagesValues(ctx, explicitPages, githubPagesObj)
				resp.Diagnostics.Append(mergeDiags...)
				if !mergeDiags.HasError() {
					plan.Pages = mergedPages
				} else {
					plan.Pages = explicitPages
				}
			} else {
				plan.Pages = explicitPages
			}
		} else {
			mergedPages, mergeDiags := r.mergePagesValues(ctx, explicitPages, types.ObjectNull(pagesObjectAttributeTypes()))
			resp.Diagnostics.Append(mergeDiags...)
			if !mergeDiags.HasError() {
				plan.Pages = mergedPages
			} else {
				plan.Pages = explicitPages
			}
		}
	} else {
		plan.Pages = types.ObjectNull(pagesObjectAttributeTypes())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *repositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Error",
			"GitHub client is not configured. Please ensure a token is provided.",
		)
		return
	}

	// Get owner, falling back to authenticated user if not set
	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	repoName := state.ID.ValueString()
	if repoName == "" {
		resp.Diagnostics.AddError(
			"Missing Repository Name",
			"The repository name (id) is required.",
		)
		return
	}

	existingHasWiki := state.HasWiki
	existingHasIssues := state.HasIssues
	existingHasProjects := state.HasProjects
	existingHasDownloads := state.HasDownloads
	existingHasDiscussions := state.HasDiscussions
	existingPages := state.Pages

	r.readRepository(ctx, owner, repoName, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !existingHasWiki.IsNull() && !existingHasWiki.IsUnknown() {
		state.HasWiki = existingHasWiki
	}
	if !existingHasIssues.IsNull() && !existingHasIssues.IsUnknown() {
		state.HasIssues = existingHasIssues
	}
	if !existingHasProjects.IsNull() && !existingHasProjects.IsUnknown() {
		state.HasProjects = existingHasProjects
	}
	if !existingHasDownloads.IsNull() && !existingHasDownloads.IsUnknown() {
		state.HasDownloads = existingHasDownloads
	}
	if !existingHasDiscussions.IsNull() && !existingHasDiscussions.IsUnknown() {
		state.HasDiscussions = existingHasDiscussions
	}

	if !existingPages.IsNull() && !existingPages.IsUnknown() {
		pages, _, err := r.client.Repositories.GetPagesInfo(ctx, owner, repoName)
		if err == nil && pages != nil {
			pagesObj, pageDiags := flattenPages(ctx, pages)
			if !pageDiags.HasError() {
				mergedPages, mergeDiags := r.mergePagesValues(ctx, existingPages, pagesObj)
				resp.Diagnostics.Append(mergeDiags...)
				if !mergeDiags.HasError() {
					state.Pages = mergedPages
				} else {
					state.Pages = existingPages
				}
			} else {
				state.Pages = existingPages
			}
		} else {
			state.Pages = existingPages
		}
	} else {
		state.Pages = types.ObjectNull(pagesObjectAttributeTypes())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *repositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state repositoryResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Error",
			"GitHub client is not configured. Please ensure a token is provided.",
		)
		return
	}

	// Get owner, falling back to authenticated user if not set
	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	if plan.Topics.IsUnknown() {
		plan.Topics = types.SetNull(types.StringType)
	}

	repoName := state.ID.ValueString()
	repoReq := &github.Repository{}

	if !plan.Description.Equal(state.Description) {
		repoReq.Description = github.String(plan.Description.ValueString())
	}

	if !plan.HomepageURL.Equal(state.HomepageURL) {
		if !plan.HomepageURL.IsNull() && plan.HomepageURL.ValueString() != "" {
			repoReq.Homepage = github.String(plan.HomepageURL.ValueString())
		} else {
			repoReq.Homepage = github.String("")
		}
	}

	if !plan.Visibility.Equal(state.Visibility) {
		visibility := plan.Visibility.ValueString()
		repoReq.Visibility = github.String(visibility)
		if visibility == "private" {
			repoReq.Private = github.Bool(true)
		} else {
			repoReq.Private = github.Bool(false)
		}
	}

	if !plan.HasIssues.Equal(state.HasIssues) {
		repoReq.HasIssues = github.Bool(plan.HasIssues.ValueBool())
	}
	if !plan.HasDiscussions.Equal(state.HasDiscussions) {
		repoReq.HasDiscussions = github.Bool(plan.HasDiscussions.ValueBool())
	}
	if !plan.HasProjects.Equal(state.HasProjects) {
		repoReq.HasProjects = github.Bool(plan.HasProjects.ValueBool())
	}
	if !plan.HasDownloads.Equal(state.HasDownloads) {
		repoReq.HasDownloads = github.Bool(plan.HasDownloads.ValueBool())
	}
	if !plan.HasWiki.Equal(state.HasWiki) {
		repoReq.HasWiki = github.Bool(plan.HasWiki.ValueBool())
	}
	if !plan.IsTemplate.Equal(state.IsTemplate) {
		repoReq.IsTemplate = github.Bool(plan.IsTemplate.ValueBool())
	}

	allowMergeCommit := state.AllowMergeCommit.ValueBool()
	allowSquashMerge := state.AllowSquashMerge.ValueBool()
	allowRebaseMerge := state.AllowRebaseMerge.ValueBool()

	if !plan.AllowMergeCommit.IsNull() && !plan.AllowMergeCommit.IsUnknown() {
		if !plan.AllowMergeCommit.Equal(state.AllowMergeCommit) {
			allowMergeCommit = plan.AllowMergeCommit.ValueBool()
			repoReq.AllowMergeCommit = github.Bool(allowMergeCommit)
		}
	}
	if !plan.AllowSquashMerge.IsNull() && !plan.AllowSquashMerge.IsUnknown() {
		if !plan.AllowSquashMerge.Equal(state.AllowSquashMerge) {
			allowSquashMerge = plan.AllowSquashMerge.ValueBool()
			repoReq.AllowSquashMerge = github.Bool(allowSquashMerge)
		}
	}
	if !plan.AllowRebaseMerge.IsNull() && !plan.AllowRebaseMerge.IsUnknown() {
		if !plan.AllowRebaseMerge.Equal(state.AllowRebaseMerge) {
			allowRebaseMerge = plan.AllowRebaseMerge.ValueBool()
			repoReq.AllowRebaseMerge = github.Bool(allowRebaseMerge)
		}
	}

	finalAllowMergeCommit := allowMergeCommit
	finalAllowSquashMerge := allowSquashMerge
	finalAllowRebaseMerge := allowRebaseMerge

	if repoReq.AllowMergeCommit != nil {
		finalAllowMergeCommit = *repoReq.AllowMergeCommit
	}
	if repoReq.AllowSquashMerge != nil {
		finalAllowSquashMerge = *repoReq.AllowSquashMerge
	}
	if repoReq.AllowRebaseMerge != nil {
		finalAllowRebaseMerge = *repoReq.AllowRebaseMerge
	}

	if !finalAllowMergeCommit && !finalAllowSquashMerge && !finalAllowRebaseMerge {
		finalAllowMergeCommit = true
		repoReq.AllowMergeCommit = github.Bool(true)
		allowMergeCommit = true
	}

	if !plan.AllowAutoMerge.IsNull() && !plan.AllowAutoMerge.IsUnknown() {
		if !plan.AllowAutoMerge.Equal(state.AllowAutoMerge) {
			repoReq.AllowAutoMerge = github.Bool(plan.AllowAutoMerge.ValueBool())
		}
	}
	if !plan.AllowUpdateBranch.IsNull() && !plan.AllowUpdateBranch.IsUnknown() {
		if !plan.AllowUpdateBranch.Equal(state.AllowUpdateBranch) {
			repoReq.AllowUpdateBranch = github.Bool(plan.AllowUpdateBranch.ValueBool())
		}
	}

	// Only set squash merge commit settings if squash merge is enabled
	// GitHub requires squash merge to be enabled to set these fields
	if allowSquashMerge {
		if !plan.SquashMergeCommitTitle.IsNull() && !plan.SquashMergeCommitTitle.IsUnknown() && plan.SquashMergeCommitTitle.ValueString() != "" {
			if !plan.SquashMergeCommitTitle.Equal(state.SquashMergeCommitTitle) {
				repoReq.SquashMergeCommitTitle = github.String(plan.SquashMergeCommitTitle.ValueString())
			}
		}
		if !plan.SquashMergeCommitMessage.IsNull() && !plan.SquashMergeCommitMessage.IsUnknown() && plan.SquashMergeCommitMessage.ValueString() != "" {
			if !plan.SquashMergeCommitMessage.Equal(state.SquashMergeCommitMessage) {
				repoReq.SquashMergeCommitMessage = github.String(plan.SquashMergeCommitMessage.ValueString())
			}
		}
	}

	if allowMergeCommit {
		hasMergeTitle := !plan.MergeCommitTitle.IsNull() && !plan.MergeCommitTitle.IsUnknown() && plan.MergeCommitTitle.ValueString() != ""
		hasMergeMessage := !plan.MergeCommitMessage.IsNull() && !plan.MergeCommitMessage.IsUnknown() && plan.MergeCommitMessage.ValueString() != ""

		if hasMergeTitle && hasMergeMessage {
			titleChanged := !plan.MergeCommitTitle.Equal(state.MergeCommitTitle)
			messageChanged := !plan.MergeCommitMessage.Equal(state.MergeCommitMessage)

			if titleChanged || messageChanged {
				repoReq.MergeCommitTitle = github.String(plan.MergeCommitTitle.ValueString())
				repoReq.MergeCommitMessage = github.String(plan.MergeCommitMessage.ValueString())
			}
		}
	}

	if !plan.DeleteBranchOnMerge.Equal(state.DeleteBranchOnMerge) {
		repoReq.DeleteBranchOnMerge = github.Bool(plan.DeleteBranchOnMerge.ValueBool())
	}

	if !plan.Archived.Equal(state.Archived) {
		repoReq.Archived = github.Bool(plan.Archived.ValueBool())
	}

	if repoReq.AllowMergeCommit != nil || repoReq.AllowSquashMerge != nil || repoReq.AllowRebaseMerge != nil {
		if !finalAllowMergeCommit && !finalAllowSquashMerge && !finalAllowRebaseMerge {
			repoReq.AllowMergeCommit = github.Bool(true)
		}
	} else {
		hasOtherChanges := repoReq.Description != nil || repoReq.Homepage != nil || repoReq.Visibility != nil ||
			repoReq.Private != nil || repoReq.HasIssues != nil || repoReq.HasDiscussions != nil ||
			repoReq.HasProjects != nil || repoReq.HasDownloads != nil || repoReq.HasWiki != nil ||
			repoReq.IsTemplate != nil

		if hasOtherChanges {
			if !allowMergeCommit && !allowSquashMerge && !allowRebaseMerge {
				allowMergeCommit = true
			}
			repoReq.AllowMergeCommit = github.Bool(allowMergeCommit)
			repoReq.AllowSquashMerge = github.Bool(allowSquashMerge)
			repoReq.AllowRebaseMerge = github.Bool(allowRebaseMerge)
		}
	}

	hasChanges := repoReq.Description != nil || repoReq.Homepage != nil ||
		repoReq.Visibility != nil || repoReq.Private != nil ||
		repoReq.HasIssues != nil || repoReq.HasDiscussions != nil ||
		repoReq.HasProjects != nil || repoReq.HasDownloads != nil ||
		repoReq.HasWiki != nil || repoReq.IsTemplate != nil ||
		repoReq.AllowMergeCommit != nil || repoReq.AllowSquashMerge != nil ||
		repoReq.AllowRebaseMerge != nil || repoReq.AllowAutoMerge != nil ||
		repoReq.AllowUpdateBranch != nil || repoReq.SquashMergeCommitTitle != nil ||
		repoReq.SquashMergeCommitMessage != nil || repoReq.MergeCommitTitle != nil ||
		repoReq.MergeCommitMessage != nil || repoReq.DeleteBranchOnMerge != nil ||
		repoReq.Archived != nil

	if hasChanges {
		_, _, err := r.client.Repositories.Edit(ctx, owner, repoName, repoReq)
		if err != nil {
			if !strings.Contains(err.Error(), "422 Privacy is already set") {
				resp.Diagnostics.AddError(
					"Error updating repository",
					fmt.Sprintf("Unable to update repository %s: %v", repoName, err),
				)
				return
			}
		}
	}

	if !plan.Topics.Equal(state.Topics) {
		if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
			topics := make([]string, 0, len(plan.Topics.Elements()))
			resp.Diagnostics.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
			if !resp.Diagnostics.HasError() {
				sort.Strings(topics)
				_, _, err := r.client.Repositories.ReplaceAllTopics(ctx, owner, repoName, topics)
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Error updating topics",
						fmt.Sprintf("Unable to update topics: %v", err),
					)
				}
			}
		} else {
			_, _, err := r.client.Repositories.ReplaceAllTopics(ctx, owner, repoName, []string{})
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error clearing topics",
					fmt.Sprintf("Unable to clear topics: %v", err),
				)
			}
		}
	}

	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		pagesChanged := false
		if state.Pages.IsNull() || state.Pages.IsUnknown() {
			pagesChanged = true
		} else {
			var planModel, stateModel pagesModel
			planDiags := plan.Pages.As(ctx, &planModel, basetypes.ObjectAsOptions{})
			stateDiags := state.Pages.As(ctx, &stateModel, basetypes.ObjectAsOptions{})
			if !planDiags.HasError() && !stateDiags.HasError() {
				if !planModel.Source.Equal(stateModel.Source) ||
					!planModel.BuildType.Equal(stateModel.BuildType) ||
					!planModel.CNAME.Equal(stateModel.CNAME) {
					pagesChanged = true
				}
			} else {
				pagesChanged = true
			}
		}

		if pagesChanged {
			diags := r.updatePages(ctx, owner, repoName, plan.Pages)
			resp.Diagnostics.Append(diags...)
		}
	}

	if !plan.VulnerabilityAlerts.Equal(state.VulnerabilityAlerts) {
		diags := r.updateVulnerabilityAlerts(ctx, owner, repoName, plan.VulnerabilityAlerts.ValueBool())
		resp.Diagnostics.Append(diags...)
	}

	planHasWiki := plan.HasWiki
	planHasIssues := plan.HasIssues
	planHasProjects := plan.HasProjects
	planHasDownloads := plan.HasDownloads
	planHasDiscussions := plan.HasDiscussions
	planPages := plan.Pages

	r.readRepository(ctx, owner, repoName, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if planHasWiki.Equal(state.HasWiki) {
		plan.HasWiki = state.HasWiki
	} else if !planHasWiki.IsNull() && !planHasWiki.IsUnknown() {
		plan.HasWiki = planHasWiki
	} else if !state.HasWiki.IsNull() && !state.HasWiki.IsUnknown() {
		plan.HasWiki = state.HasWiki
	}

	if planHasIssues.Equal(state.HasIssues) {
		plan.HasIssues = state.HasIssues
	} else if !planHasIssues.IsNull() && !planHasIssues.IsUnknown() {
		plan.HasIssues = planHasIssues
	} else if !state.HasIssues.IsNull() && !state.HasIssues.IsUnknown() {
		plan.HasIssues = state.HasIssues
	}

	if planHasProjects.Equal(state.HasProjects) {
		plan.HasProjects = state.HasProjects
	} else if !planHasProjects.IsNull() && !planHasProjects.IsUnknown() {
		plan.HasProjects = planHasProjects
	} else if !state.HasProjects.IsNull() && !state.HasProjects.IsUnknown() {
		plan.HasProjects = state.HasProjects
	}

	if planHasDownloads.Equal(state.HasDownloads) {
		plan.HasDownloads = state.HasDownloads
	} else if !planHasDownloads.IsNull() && !planHasDownloads.IsUnknown() {
		plan.HasDownloads = planHasDownloads
	} else if !state.HasDownloads.IsNull() && !state.HasDownloads.IsUnknown() {
		plan.HasDownloads = state.HasDownloads
	}

	if planHasDiscussions.Equal(state.HasDiscussions) {
		plan.HasDiscussions = state.HasDiscussions
	} else if !planHasDiscussions.IsNull() && !planHasDiscussions.IsUnknown() {
		plan.HasDiscussions = planHasDiscussions
	} else if !state.HasDiscussions.IsNull() && !state.HasDiscussions.IsUnknown() {
		plan.HasDiscussions = state.HasDiscussions
	}

	if planPages.IsNull() || planPages.IsUnknown() {
		plan.Pages = types.ObjectNull(pagesObjectAttributeTypes())
	} else {
		pages, _, err := r.client.Repositories.GetPagesInfo(ctx, owner, repoName)
		if err == nil && pages != nil {
			githubPagesObj, pageDiags := flattenPages(ctx, pages)
			if !pageDiags.HasError() {
				mergedPages, mergeDiags := r.mergePagesValues(ctx, planPages, githubPagesObj)
				resp.Diagnostics.Append(mergeDiags...)
				if !mergeDiags.HasError() {
					plan.Pages = mergedPages
				} else {
					plan.Pages = planPages
				}
			} else {
				plan.Pages = planPages
			}
		} else {
			mergedPages, mergeDiags := r.mergePagesValues(ctx, planPages, types.ObjectNull(pagesObjectAttributeTypes()))
			resp.Diagnostics.Append(mergeDiags...)
			if !mergeDiags.HasError() {
				plan.Pages = mergedPages
			} else {
				plan.Pages = planPages
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *repositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Error",
			"GitHub client is not configured. Please ensure a token is provided.",
		)
		return
	}

	// Get owner, falling back to authenticated user if not set
	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	repoName := state.ID.ValueString()
	archiveOnDestroy := state.ArchiveOnDestroy.ValueBool()
	if archiveOnDestroy {
		if state.Archived.ValueBool() {
			log.Printf("[DEBUG] Repository already archived, nothing to do on delete: %s/%s", owner, repoName)
			return
		}

		repoReq := &github.Repository{
			Archived: github.Bool(true),
		}
		log.Printf("[DEBUG] Archiving repository on delete: %s/%s", owner, repoName)
		_, _, err := r.client.Repositories.Edit(ctx, owner, repoName, repoReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error archiving repository",
				fmt.Sprintf("Unable to archive repository %s: %v", repoName, err),
			)
		}
		return
	}

	log.Printf("[DEBUG] Deleting repository: %s/%s", owner, repoName)
	_, err = r.client.Repositories.Delete(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting repository",
			fmt.Sprintf("Unable to delete repository %s: %v", repoName, err),
		)
		return
	}
}

func (r *repositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	var repoName string

	if len(parts) == 2 {
		repoName = parts[1]
	} else if len(parts) == 1 {
		repoName = parts[0]
		_, err := r.getOwner(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Import ID must be in format 'owner/repo' or 'repo' (when provider-level owner is configured or authentication is available). Error: %v", err),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'owner/repo' or 'repo'.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), repoName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), repoName)...)
}

func (r *repositoryResource) getOwner(ctx context.Context) (string, error) {
	if r.owner != "" {
		return r.owner, nil
	}

	user, _, err := r.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and unable to fetch authenticated user: %v", err)
	}
	if user == nil || user.Login == nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and authenticated user information is unavailable")
	}
	return user.GetLogin(), nil
}

func (r *repositoryResource) readRepository(ctx context.Context, owner, repoName string, model *repositoryResourceModel, diags *diag.Diagnostics) {
	repo, _, err := r.client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		diags.AddError(
			"Error reading repository",
			fmt.Sprintf("Unable to read repository %s/%s: %v", owner, repoName, err),
		)
		return
	}

	model.ID = types.StringValue(repo.GetName())
	model.Name = types.StringValue(repo.GetName())

	fullName := repo.GetFullName()
	if fullName == "" {
		fullName = fmt.Sprintf("%s/%s", owner, repo.GetName())
	}
	model.FullName = types.StringValue(fullName)

	model.Description = types.StringValue(repo.GetDescription())

	homepage := repo.GetHomepage()
	if homepage == "" {
		model.HomepageURL = types.StringNull()
	} else {
		model.HomepageURL = types.StringValue(homepage)
	}

	visibility := repo.GetVisibility()
	if visibility == "" {
		visibility = "public"
	}
	model.Visibility = types.StringValue(visibility)

	model.HasIssues = types.BoolValue(repo.GetHasIssues())
	model.HasDiscussions = types.BoolValue(repo.GetHasDiscussions())
	model.HasProjects = types.BoolValue(repo.GetHasProjects())
	model.HasDownloads = types.BoolValue(repo.GetHasDownloads())
	model.HasWiki = types.BoolValue(repo.GetHasWiki())
	model.IsTemplate = types.BoolValue(repo.GetIsTemplate())
	model.AllowMergeCommit = types.BoolValue(repo.GetAllowMergeCommit())
	model.AllowSquashMerge = types.BoolValue(repo.GetAllowSquashMerge())
	model.AllowRebaseMerge = types.BoolValue(repo.GetAllowRebaseMerge())
	model.AllowAutoMerge = types.BoolValue(repo.GetAllowAutoMerge())
	model.AllowUpdateBranch = types.BoolValue(repo.GetAllowUpdateBranch())
	model.SquashMergeCommitTitle = types.StringValue(repo.GetSquashMergeCommitTitle())
	model.SquashMergeCommitMessage = types.StringValue(repo.GetSquashMergeCommitMessage())
	model.MergeCommitTitle = types.StringValue(repo.GetMergeCommitTitle())
	model.MergeCommitMessage = types.StringValue(repo.GetMergeCommitMessage())
	model.DeleteBranchOnMerge = types.BoolValue(repo.GetDeleteBranchOnMerge())
	model.Archived = types.BoolValue(repo.GetArchived())

	defaultBranch := repo.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	model.DefaultBranch = types.StringValue(defaultBranch)

	htmlURL := repo.GetHTMLURL()
	if htmlURL == "" {
		htmlURL = fmt.Sprintf("https://github.com/%s/%s", owner, repo.GetName())
	}
	model.HTMLURL = types.StringValue(htmlURL)

	nodeID := repo.GetNodeID()
	model.NodeID = types.StringValue(nodeID)

	model.RepoID = types.Int64Value(repo.GetID())

	if len(repo.Topics) > 0 {
		sortedTopics := make([]string, len(repo.Topics))
		copy(sortedTopics, repo.Topics)
		sort.Strings(sortedTopics)

		topics := make([]types.String, len(sortedTopics))
		for i, topic := range sortedTopics {
			topics[i] = types.StringValue(topic)
		}
		topicsSet, topicDiags := types.SetValueFrom(ctx, types.StringType, topics)
		diags.Append(topicDiags...)
		model.Topics = topicsSet
	} else {
		model.Topics = types.SetNull(types.StringType)
	}

	model.Pages = types.ObjectNull(pagesObjectAttributeTypes())

	_, resp, err := r.client.Repositories.GetVulnerabilityAlerts(ctx, owner, repoName)
	if err != nil {
		if errResp, ok := err.(*github.ErrorResponse); ok && errResp != nil && errResp.Response != nil && errResp.Response.StatusCode == http.StatusNotFound {
			model.VulnerabilityAlerts = types.BoolValue(false)
		} else {
			diags.AddWarning(
				"Error reading vulnerability alerts",
				fmt.Sprintf("Unable to read vulnerability alerts: %v", err),
			)
			model.VulnerabilityAlerts = types.BoolNull()
		}
	} else {
		enabled := resp != nil && resp.StatusCode == http.StatusNoContent
		model.VulnerabilityAlerts = types.BoolValue(enabled)
	}
}

func (r *repositoryResource) mergePagesValues(ctx context.Context, planPages, githubPages types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if githubPages.IsNull() || githubPages.IsUnknown() {
		if planPages.IsNull() || planPages.IsUnknown() {
			return types.ObjectNull(pagesObjectAttributeTypes()), diags
		}

		var planModel pagesModel
		diags.Append(planPages.As(ctx, &planModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(pagesObjectAttributeTypes()), diags
		}

		var source types.Object
		if planModel.Source.IsNull() || planModel.Source.IsUnknown() {
			source = types.ObjectNull(pagesSourceObjectAttributeTypes())
		} else {
			var sourceModel pagesSourceModel
			sourceDiags := planModel.Source.As(ctx, &sourceModel, basetypes.ObjectAsOptions{})
			if sourceDiags.HasError() {
				source = types.ObjectNull(pagesSourceObjectAttributeTypes())
			} else {
				branch := sourceModel.Branch
				if branch.IsNull() || branch.IsUnknown() {
					branch = types.StringNull()
				}
				path := sourceModel.Path
				pathValue := path.ValueString()
				if path.IsNull() || path.IsUnknown() || pathValue == "" {
					path = types.StringValue("/")
				}
				sourceAttrs := map[string]attr.Value{
					"branch": branch,
					"path":   path,
				}
				sourceObj, sourceObjDiags := types.ObjectValue(pagesSourceObjectAttributeTypes(), sourceAttrs)
				if sourceObjDiags.HasError() {
					source = types.ObjectNull(pagesSourceObjectAttributeTypes())
				} else {
					source = sourceObj
				}
			}
		}

		buildType := planModel.BuildType
		if buildType.IsUnknown() {
			buildType = types.StringNull()
		}
		cname := planModel.CNAME
		if cname.IsUnknown() {
			cname = types.StringNull()
		}

		attrs := map[string]attr.Value{
			"source":     source,
			"build_type": buildType,
			"cname":      cname,
			"custom_404": types.BoolNull(),
			"html_url":   types.StringNull(),
			"status":     types.StringNull(),
			"url":        types.StringNull(),
		}

		mergedObj, objDiags := types.ObjectValue(pagesObjectAttributeTypes(), attrs)
		diags.Append(objDiags...)
		return mergedObj, diags
	}

	var planModel pagesModel
	var githubModel pagesModel

	if !planPages.IsNull() && !planPages.IsUnknown() {
		diags.Append(planPages.As(ctx, &planModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(pagesObjectAttributeTypes()), diags
		}
	}

	diags.Append(githubPages.As(ctx, &githubModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return types.ObjectNull(pagesObjectAttributeTypes()), diags
	}

	source := planModel.Source
	if source.IsNull() || source.IsUnknown() {
		source = githubModel.Source
	}
	buildType := planModel.BuildType
	if buildType.IsNull() || buildType.IsUnknown() {
		buildType = githubModel.BuildType
	}
	cname := planModel.CNAME
	if cname.IsNull() || cname.IsUnknown() {
		cname = githubModel.CNAME
	}

	mergedAttrs := map[string]attr.Value{
		"source":     source,
		"build_type": buildType,
		"cname":      cname,
		"custom_404": githubModel.Custom404,
		"html_url":   githubModel.HTMLURL,
		"status":     githubModel.Status,
		"url":        githubModel.URL,
	}

	mergedObj, objDiags := types.ObjectValue(pagesObjectAttributeTypes(), mergedAttrs)
	diags.Append(objDiags...)
	return mergedObj, diags
}

func (r *repositoryResource) updatePages(ctx context.Context, owner, repoName string, pagesObj types.Object) diag.Diagnostics {
	var diags diag.Diagnostics

	if pagesObj.IsNull() || pagesObj.IsUnknown() {
		log.Printf("[DEBUG] Pages configuration cannot be removed via API")
		return diags
	}

	var pagesModel pagesModel
	diags.Append(pagesObj.As(ctx, &pagesModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}

	pagesUpdate := &github.PagesUpdate{}

	if !pagesModel.CNAME.IsNull() {
		cname := pagesModel.CNAME.ValueString()
		if cname != "" {
			pagesUpdate.CNAME = github.String(cname)
		}
	}

	if !pagesModel.BuildType.IsNull() {
		buildType := pagesModel.BuildType.ValueString()
		if buildType != "" {
			pagesUpdate.BuildType = github.String(buildType)
		}
	}

	if !pagesModel.Source.IsNull() {
		var sourceModel pagesSourceModel
		diags.Append(pagesModel.Source.As(ctx, &sourceModel, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			branch := sourceModel.Branch.ValueString()
			path := sourceModel.Path.ValueString()
			if path == "" || path == "/" {
				path = ""
			}
			pagesUpdate.Source = &github.PagesSource{
				Branch: github.String(branch),
				Path:   github.String(path),
			}
		}
	}

	_, err := r.client.Repositories.UpdatePages(ctx, owner, repoName, pagesUpdate)
	if err != nil {
		if errResp, ok := err.(*github.ErrorResponse); ok && errResp != nil && errResp.Response != nil && errResp.Response.StatusCode == http.StatusNotFound {
			log.Printf("[DEBUG] Pages not yet available for repository %s/%s (404): %v. Pages will be configured once the repository has content.", owner, repoName, err)
		} else {
			diags.AddWarning(
				"Error updating Pages",
				fmt.Sprintf("Unable to update Pages configuration: %v. Pages may not be available until the repository has content.", err),
			)
		}
	}

	return diags
}

func (r *repositoryResource) updateVulnerabilityAlerts(ctx context.Context, owner, repoName string, enabled bool) diag.Diagnostics {
	var diags diag.Diagnostics

	if enabled {
		_, err := r.client.Repositories.EnableVulnerabilityAlerts(ctx, owner, repoName)
		if err != nil {
			diags.AddWarning(
				"Error enabling vulnerability alerts",
				fmt.Sprintf("Unable to enable vulnerability alerts: %v", err),
			)
		}
	} else {
		_, err := r.client.Repositories.DisableVulnerabilityAlerts(ctx, owner, repoName)
		if err != nil {
			diags.AddWarning(
				"Error disabling vulnerability alerts",
				fmt.Sprintf("Unable to disable vulnerability alerts: %v", err),
			)
		}
	}

	return diags
}
