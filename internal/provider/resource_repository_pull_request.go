package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &repositoryPullRequestResource{}
	_ resource.ResourceWithConfigure   = &repositoryPullRequestResource{}
	_ resource.ResourceWithImportState = &repositoryPullRequestResource{}
)

// NewRepositoryPullRequestResource is a helper function to simplify the provider implementation.
func NewRepositoryPullRequestResource() resource.Resource {
	return &repositoryPullRequestResource{}
}

// repositoryPullRequestResource is the resource implementation.
type repositoryPullRequestResource struct {
	client *github.Client
	owner  string
}

// repositoryPullRequestResourceModel maps the resource schema data.
type repositoryPullRequestResourceModel struct {
	Repository          types.String `tfsdk:"repository"`
	BaseRef             types.String `tfsdk:"base_ref"`
	HeadRef             types.String `tfsdk:"head_ref"`
	Title               types.String `tfsdk:"title"`
	Body                types.String `tfsdk:"body"`
	MergeWhenReady      types.Bool   `tfsdk:"merge_when_ready"`
	MergeMethod         types.String `tfsdk:"merge_method"`
	WaitForChecks       types.Bool   `tfsdk:"wait_for_checks"`
	AutoDeleteBranch    types.Bool   `tfsdk:"auto_delete_branch"`
	MaintainerCanModify types.Bool   `tfsdk:"maintainer_can_modify"`
	BaseSHA             types.String `tfsdk:"base_sha"`
	HeadSHA             types.String `tfsdk:"head_sha"`
	Number              types.Int64  `tfsdk:"number"`
	State               types.String `tfsdk:"state"`
	Merged              types.Bool   `tfsdk:"merged"`
	MergedAt            types.String `tfsdk:"merged_at"`
	MergeCommitSHA      types.String `tfsdk:"merge_commit_sha"`
	ID                  types.String `tfsdk:"id"`
}

// Metadata returns the resource type name.
func (r *repositoryPullRequestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_pull_request_auto_merge"
}

// Schema defines the schema for the resource.
func (r *repositoryPullRequestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a GitHub pull request with optional auto-merge capabilities. Supports multiple files through branch-based commits.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"base_ref": schema.StringAttribute{
				Description: "The base branch name (e.g., 'main', 'develop').",
				Required:    true,
			},
			"head_ref": schema.StringAttribute{
				Description: "The head branch name (e.g., 'feature-branch').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title of the pull request.",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "The body/description of the pull request.",
				Optional:    true,
			},
			"merge_when_ready": schema.BoolAttribute{
				Description: "Wait for all checks and approvals to pass, then automatically merge.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"merge_method": schema.StringAttribute{
				Description: "The merge method to use when auto-merging. Options: 'merge', 'squash', 'rebase'. Defaults to 'merge'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("merge"),
			},
			"wait_for_checks": schema.BoolAttribute{
				Description: "Wait for CI checks to pass before merging. Only applies when 'merge_when_ready' is true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"auto_delete_branch": schema.BoolAttribute{
				Description: "Automatically delete the head branch after merge.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"maintainer_can_modify": schema.BoolAttribute{
				Description: "Allow maintainers to modify the pull request.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"base_sha": schema.StringAttribute{
				Description: "The SHA of the base branch.",
				Computed:    true,
			},
			"head_sha": schema.StringAttribute{
				Description: "The SHA of the head branch.",
				Computed:    true,
			},
			"number": schema.Int64Attribute{
				Description: "The pull request number.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the pull request (open, closed, merged).",
				Computed:    true,
			},
			"merged": schema.BoolAttribute{
				Description: "Whether the pull request has been merged.",
				Computed:    true,
			},
			"merged_at": schema.StringAttribute{
				Description: "The timestamp when the pull request was merged.",
				Computed:    true,
			},
			"merge_commit_sha": schema.StringAttribute{
				Description: "The SHA of the merge commit.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository:number).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure enables provider-level data or clients to be set in the
// provider-defined resource type.
func (r *repositoryPullRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

// Create creates the resource and sets the initial Terraform state.
func (r *repositoryPullRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryPullRequestResourceModel

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

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	repoName := plan.Repository.ValueString()
	baseRef := plan.BaseRef.ValueString()
	headRef := plan.HeadRef.ValueString()
	title := plan.Title.ValueString()

	if baseRef == headRef {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			fmt.Sprintf("Base branch '%s' and head branch '%s' cannot be the same. There must be a difference to create a pull request.", baseRef, headRef),
		)
		return
	}

	existingPR, err := r.findExistingPR(ctx, owner, repoName, baseRef, headRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking for existing pull request",
			fmt.Sprintf("Unable to check for existing pull request in repository %s/%s: %v", owner, repoName, err),
		)
		return
	}

	var pr *github.PullRequest
	if existingPR != nil {
		if existingPR.GetState() == "closed" && !existingPR.GetMerged() {
			// Reopen the closed PR
			log.Printf("[INFO] Reopening closed pull request #%d from '%s' to '%s' in repository %s/%s", existingPR.GetNumber(), headRef, baseRef, owner, repoName)
			update := &github.PullRequest{
				State: github.String("open"),
			}
			reopenedPR, _, err := r.client.PullRequests.Edit(ctx, owner, repoName, existingPR.GetNumber(), update)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reopening pull request",
					fmt.Sprintf("Unable to reopen pull request #%d in repository %s/%s: %v", existingPR.GetNumber(), owner, repoName, err),
				)
				return
			}
			pr = reopenedPR
		} else {
			// Adopt the existing PR - this handles the case where PR was created outside Terraform
			// or if terraform apply is run again after the PR was already created
			log.Printf("[INFO] Adopting existing pull request #%d from '%s' to '%s' in repository %s/%s", existingPR.GetNumber(), headRef, baseRef, owner, repoName)
			pr = existingPR
		}
		plan.ID = types.StringValue(fmt.Sprintf("%s:%d", repoName, pr.GetNumber()))
		plan.Number = types.Int64Value(int64(pr.GetNumber()))
	} else {
		// No existing PR found, create a new one
		baseSHA, headSHA, err := r.getBranchSHAs(ctx, owner, repoName, baseRef, headRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error checking branch differences",
				fmt.Sprintf("Unable to get branch information: %v", err),
			)
			return
		}

		if baseSHA == headSHA {
			resp.Diagnostics.AddError(
				"No Differences",
				fmt.Sprintf("Branches '%s' and '%s' are at the same commit (SHA: %s). There are no changes to create a pull request.", headRef, baseRef, headSHA),
			)
			return
		}

		newPR := &github.NewPullRequest{
			Title:               github.String(title),
			Head:                github.String(headRef),
			Base:                github.String(baseRef),
			MaintainerCanModify: github.Bool(plan.MaintainerCanModify.ValueBool()),
		}

		if !plan.Body.IsNull() && !plan.Body.IsUnknown() {
			newPR.Body = github.String(plan.Body.ValueString())
		}

		pr, _, err = r.client.PullRequests.Create(ctx, owner, repoName, newPR)
		if err != nil {
			var ghErr *github.ErrorResponse
			if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusUnprocessableEntity {
				if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "No commits between") {
					resp.Diagnostics.AddError(
						"Pull Request Already Exists or No Changes",
						fmt.Sprintf("Unable to create pull request: %v. A pull request may already exist, or there are no commits between '%s' and '%s'.", err, headRef, baseRef),
					)
					return
				}
			}
			resp.Diagnostics.AddError(
				"Error creating pull request",
				fmt.Sprintf("Unable to create pull request in repository %s/%s: %v", owner, repoName, err),
			)
			return
		}

		plan.ID = types.StringValue(fmt.Sprintf("%s:%d", repoName, pr.GetNumber()))
		plan.Number = types.Int64Value(int64(pr.GetNumber()))
	}

	// Only attempt auto-merge if PR is open and not already merged
	if pr.GetState() == "open" && !pr.GetMerged() {
		if plan.MergeWhenReady.ValueBool() {
			if err := r.handleAutoMerge(ctx, owner, repoName, pr.GetNumber(), &plan, &resp.Diagnostics); err != nil {
				resp.Diagnostics.AddError(
					"Error setting up auto-merge",
					fmt.Sprintf("Unable to set up auto-merge for pull request #%d: %v", pr.GetNumber(), err),
				)
				return
			}
		}
	}

	r.readPullRequest(ctx, owner, repoName, pr.GetNumber(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *repositoryPullRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryPullRequestResourceModel

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

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	id := state.ID.ValueString()
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:number'.", id),
		)
		return
	}

	repoName := parts[0]
	numberStr := parts[1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid PR Number",
			fmt.Sprintf("Invalid PR number in ID: %s", numberStr),
		)
		return
	}

	r.readPullRequest(ctx, owner, repoName, number, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.State.ValueString() == "closed" && !state.Merged.ValueBool() {
		log.Printf("[INFO] Pull request #%d is closed but not merged, removing from state", number)
		state.ID = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *repositoryPullRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state repositoryPullRequestResourceModel

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

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	id := state.ID.ValueString()
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:number'.", id),
		)
		return
	}

	repoName := parts[0]
	numberStr := parts[1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid PR Number",
			fmt.Sprintf("Invalid PR number in ID: %s", numberStr),
		)
		return
	}

	update := &github.PullRequest{
		Title:               github.String(plan.Title.ValueString()),
		MaintainerCanModify: github.Bool(plan.MaintainerCanModify.ValueBool()),
	}

	if !plan.Body.IsNull() && !plan.Body.IsUnknown() {
		update.Body = github.String(plan.Body.ValueString())
	}

	if !plan.BaseRef.Equal(state.BaseRef) {
		update.Base = &github.PullRequestBranch{
			Ref: github.String(plan.BaseRef.ValueString()),
		}
	}

	_, _, err = r.client.PullRequests.Edit(ctx, owner, repoName, number, update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating pull request",
			fmt.Sprintf("Unable to update pull request #%d in repository %s/%s: %v", number, owner, repoName, err),
		)
		return
	}

	if !plan.MergeWhenReady.Equal(state.MergeWhenReady) && plan.MergeWhenReady.ValueBool() {
		if err := r.handleAutoMerge(ctx, owner, repoName, number, &plan, &resp.Diagnostics); err != nil {
			resp.Diagnostics.AddError(
				"Error setting up auto-merge",
				fmt.Sprintf("Unable to set up auto-merge for pull request #%d: %v", number, err),
			)
			return
		}
	}

	r.readPullRequest(ctx, owner, repoName, number, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *repositoryPullRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryPullRequestResourceModel

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

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v. Please set provider-level `owner` configuration or ensure authentication is working.", err),
		)
		return
	}

	id := state.ID.ValueString()
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:number'.", id),
		)
		return
	}

	repoName := parts[0]
	numberStr := parts[1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid PR Number",
			fmt.Sprintf("Invalid PR number in ID: %s", numberStr),
		)
		return
	}

	pr, _, err := r.client.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
			log.Printf("[INFO] Pull request #%d not found, assuming already deleted", number)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading pull request",
			fmt.Sprintf("Unable to read pull request #%d from repository %s/%s: %v", number, owner, repoName, err),
		)
		return
	}

	// If PR is already merged, don't close it - just cleanup
	if pr.GetMerged() {
		log.Printf("[INFO] Pull request #%d is already merged, skipping close operation", number)
		return
	}

	// Close the PR if it's still open
	if pr.GetState() == "open" {
		update := &github.PullRequest{State: github.String("closed")}
		_, _, err = r.client.PullRequests.Edit(ctx, owner, repoName, number, update)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error closing pull request",
				fmt.Sprintf("Unable to close pull request #%d in repository %s/%s: %v", number, owner, repoName, err),
			)
			return
		}
		log.Printf("[INFO] Closed pull request #%d", number)
	} else {
		log.Printf("[INFO] Pull request #%d is already closed", number)
	}
}

// ImportState imports the resource.
func (r *repositoryPullRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'repository:number'.",
		)
		return
	}

	repoName := parts[0]
	numberStr := parts[1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid PR Number",
			fmt.Sprintf("Invalid PR number: %s", numberStr),
		)
		return
	}

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v", err),
		)
		return
	}

	pr, _, err := r.client.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing pull request",
			fmt.Sprintf("Unable to read pull request #%d from repository %s/%s: %v", number, owner, repoName, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%d", repoName, pr.GetNumber()))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), repoName)...)
}

func (r *repositoryPullRequestResource) getOwner(ctx context.Context) (string, error) {
	if r.owner != "" {
		return r.owner, nil
	}

	user, _, err := r.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("unable to get authenticated user: %w", err)
	}
	return user.GetLogin(), nil
}

func (r *repositoryPullRequestResource) readPullRequest(ctx context.Context, owner, repoName string, number int, model *repositoryPullRequestResourceModel, diags *diag.Diagnostics) {
	pr, _, err := r.client.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
			log.Printf("[INFO] Pull request #%d not found, removing from state", number)
			model.ID = types.StringValue("")
			return
		}
		diags.AddError(
			"Error reading pull request",
			fmt.Sprintf("Unable to read pull request #%d from repository %s/%s: %v", number, owner, repoName, err),
		)
		return
	}

	model.Repository = types.StringValue(repoName)
	model.Number = types.Int64Value(int64(pr.GetNumber()))
	model.Title = types.StringValue(pr.GetTitle())
	model.Body = types.StringValue(pr.GetBody())
	model.State = types.StringValue(pr.GetState())
	model.Merged = types.BoolValue(pr.GetMerged())
	model.MaintainerCanModify = types.BoolValue(pr.GetMaintainerCanModify())

	mergedAt := pr.GetMergedAt()
	if !mergedAt.IsZero() && pr.GetMerged() {
		model.MergedAt = types.StringValue(mergedAt.Format(time.RFC3339))
	} else {
		model.MergedAt = types.StringNull()
	}

	mergeCommitSHA := pr.GetMergeCommitSHA()
	if mergeCommitSHA != "" && pr.GetMerged() {
		model.MergeCommitSHA = types.StringValue(mergeCommitSHA)
	} else {
		model.MergeCommitSHA = types.StringNull()
	}

	if head := pr.GetHead(); head != nil {
		model.HeadRef = types.StringValue(head.GetRef())
		model.HeadSHA = types.StringValue(head.GetSHA())
	}

	if base := pr.GetBase(); base != nil {
		model.BaseRef = types.StringValue(base.GetRef())
		model.BaseSHA = types.StringValue(base.GetSHA())
	}
}

func (r *repositoryPullRequestResource) handleAutoMerge(ctx context.Context, owner, repoName string, number int, plan *repositoryPullRequestResourceModel, diags *diag.Diagnostics) error {
	if plan.MergeWhenReady.ValueBool() {
		return r.mergeWhenReady(ctx, owner, repoName, number, plan, diags)
	}

	return nil
}

func (r *repositoryPullRequestResource) mergeWhenReady(ctx context.Context, owner, repoName string, number int, plan *repositoryPullRequestResourceModel, _ *diag.Diagnostics) error {
	maxAttempts := 30
	attempt := 0

	for attempt < maxAttempts {
		pr, _, err := r.client.PullRequests.Get(ctx, owner, repoName, number)
		if err != nil {
			return fmt.Errorf("unable to get pull request: %w", err)
		}

		if pr.GetState() != "open" {
			if pr.GetMerged() {
				return nil
			}
			return fmt.Errorf("pull request is not open")
		}

		// Check mergeability - GitHub API returns *bool (nil = not computed yet, false = not mergeable, true = mergeable)
		mergeablePtr := pr.Mergeable
		if mergeablePtr == nil {
			log.Printf("[DEBUG] PR mergeability not yet computed, waiting... (attempt %d/%d)", attempt+1, maxAttempts)
			attempt++
			time.Sleep(5 * time.Second)
			continue
		}

		if !*mergeablePtr {
			log.Printf("[DEBUG] PR is not mergeable (conflicts or other issues), waiting... (attempt %d/%d)", attempt+1, maxAttempts)
			attempt++
			time.Sleep(5 * time.Second)
			continue
		}

		// PR is mergeable, proceed with merge
		if plan.WaitForChecks.ValueBool() {
			if err := r.waitForChecks(ctx, owner, repoName, number); err != nil {
				log.Printf("[DEBUG] Checks not ready, retrying... (attempt %d/%d)", attempt+1, maxAttempts)
				attempt++
				time.Sleep(5 * time.Second)
				continue
			}
		}

		_, _, err = r.client.PullRequests.CreateReview(ctx, owner, repoName, number, &github.PullRequestReviewRequest{
			Event: github.String("APPROVE"),
			Body:  github.String("Auto-approved by Terraform"),
		})
		if err != nil {
			log.Printf("[WARN] Failed to approve PR: %v", err)
		}

		mergeMethod := plan.MergeMethod.ValueString()
		if mergeMethod == "" {
			mergeMethod = "merge"
		}

		// Check repository merge settings to validate merge method
		repo, _, err := r.client.Repositories.Get(ctx, owner, repoName)
		if err == nil && repo != nil {
			// Validate merge method against repository settings
			switch mergeMethod {
			case "squash":
				if !repo.GetAllowSquashMerge() {
					// Fall back to merge if squash not allowed
					if repo.GetAllowMergeCommit() {
						log.Printf("[WARN] Squash merge not allowed, falling back to merge commit")
						mergeMethod = "merge"
					} else if repo.GetAllowRebaseMerge() {
						log.Printf("[WARN] Squash merge not allowed, falling back to rebase")
						mergeMethod = "rebase"
					} else {
						return fmt.Errorf("squash merge is not allowed on this repository and no alternative merge methods are enabled")
					}
				}
			case "rebase":
				if !repo.GetAllowRebaseMerge() {
					// Fall back to merge if rebase not allowed
					if repo.GetAllowMergeCommit() {
						log.Printf("[WARN] Rebase merge not allowed, falling back to merge commit")
						mergeMethod = "merge"
					} else if repo.GetAllowSquashMerge() {
						log.Printf("[WARN] Rebase merge not allowed, falling back to squash")
						mergeMethod = "squash"
					} else {
						return fmt.Errorf("rebase merge is not allowed on this repository and no alternative merge methods are enabled")
					}
				}
			case "merge":
				if !repo.GetAllowMergeCommit() {
					// Fall back to squash if merge not allowed
					if repo.GetAllowSquashMerge() {
						log.Printf("[WARN] Merge commit not allowed, falling back to squash")
						mergeMethod = "squash"
					} else if repo.GetAllowRebaseMerge() {
						log.Printf("[WARN] Merge commit not allowed, falling back to rebase")
						mergeMethod = "rebase"
					} else {
						return fmt.Errorf("merge commit is not allowed on this repository and no alternative merge methods are enabled")
					}
				}
			}
		}

		_, _, err = r.client.PullRequests.Merge(ctx, owner, repoName, number, "", &github.PullRequestOptions{
			MergeMethod: mergeMethod,
		})
		if err != nil {
			// If merge fails due to method not allowed, try to find an allowed method
			var ghErr *github.ErrorResponse
			if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusMethodNotAllowed {
				if repo != nil {
					// Try alternative methods
					if mergeMethod != "merge" && repo.GetAllowMergeCommit() {
						log.Printf("[WARN] %s merge failed, trying merge commit", mergeMethod)
						_, _, err = r.client.PullRequests.Merge(ctx, owner, repoName, number, "", &github.PullRequestOptions{
							MergeMethod: "merge",
						})
					} else if mergeMethod != "squash" && repo.GetAllowSquashMerge() {
						log.Printf("[WARN] %s merge failed, trying squash", mergeMethod)
						_, _, err = r.client.PullRequests.Merge(ctx, owner, repoName, number, "", &github.PullRequestOptions{
							MergeMethod: "squash",
						})
					} else if mergeMethod != "rebase" && repo.GetAllowRebaseMerge() {
						log.Printf("[WARN] %s merge failed, trying rebase", mergeMethod)
						_, _, err = r.client.PullRequests.Merge(ctx, owner, repoName, number, "", &github.PullRequestOptions{
							MergeMethod: "rebase",
						})
					}
				}
			}
			if err != nil {
				return fmt.Errorf("unable to merge pull request: %w", err)
			}
		}

		log.Printf("[INFO] Successfully merged pull request #%d", number)

		if plan.AutoDeleteBranch.ValueBool() {
			headRef := plan.HeadRef.ValueString()
			ref := fmt.Sprintf("refs/heads/%s", headRef)
			_, err = r.client.Git.DeleteRef(ctx, owner, repoName, ref)
			if err != nil {
				log.Printf("[WARN] Failed to delete branch %s: %v", headRef, err)
			} else {
				log.Printf("[INFO] Deleted branch %s", headRef)
			}
		}

		return nil
	}

	return fmt.Errorf("pull request not ready to merge after %d attempts", maxAttempts)
}

func (r *repositoryPullRequestResource) findExistingPR(ctx context.Context, owner, repoName, baseRef, headRef string) (*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:       "all",
		Head:        fmt.Sprintf("%s:%s", owner, headRef),
		Base:        baseRef,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	prs, _, err := r.client.PullRequests.List(ctx, owner, repoName, opts)
	if err != nil {
		return nil, err
	}

	for _, pr := range prs {
		if pr.GetBase().GetRef() == baseRef && pr.GetHead().GetRef() == headRef {
			return pr, nil
		}
	}

	return nil, nil
}

func (r *repositoryPullRequestResource) getBranchSHAs(ctx context.Context, owner, repoName, baseRef, headRef string) (string, string, error) {
	baseRefFull := fmt.Sprintf("refs/heads/%s", baseRef)
	headRefFull := fmt.Sprintf("refs/heads/%s", headRef)

	baseRefObj, _, err := r.client.Git.GetRef(ctx, owner, repoName, baseRefFull)
	if err != nil {
		return "", "", fmt.Errorf("unable to get base branch %s: %w", baseRef, err)
	}

	headRefObj, _, err := r.client.Git.GetRef(ctx, owner, repoName, headRefFull)
	if err != nil {
		return "", "", fmt.Errorf("unable to get head branch %s: %w", headRef, err)
	}

	baseSHA := baseRefObj.GetObject().GetSHA()
	headSHA := headRefObj.GetObject().GetSHA()

	return baseSHA, headSHA, nil
}

func (r *repositoryPullRequestResource) waitForChecks(ctx context.Context, owner, repoName string, number int) error {
	maxAttempts := 60
	attempt := 0

	for attempt < maxAttempts {
		pr, _, err := r.client.PullRequests.Get(ctx, owner, repoName, number)
		if err != nil {
			return fmt.Errorf("unable to get pull request: %w", err)
		}

		headSHA := pr.GetHead().GetSHA()
		statuses, _, err := r.client.Repositories.ListStatuses(ctx, owner, repoName, headSHA, nil)
		if err != nil {
			return fmt.Errorf("unable to get status checks: %w", err)
		}

		allPassed := true
		for _, status := range statuses {
			state := status.GetState()
			if state == "pending" {
				allPassed = false
				break
			}
			if state == "error" || state == "failure" {
				return fmt.Errorf("status check %s failed", status.GetContext())
			}
		}

		if allPassed && len(statuses) > 0 {
			return nil
		}

		if len(statuses) == 0 {
			log.Printf("[DEBUG] No status checks found, proceeding")
			return nil
		}

		attempt++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("status checks did not complete after %d attempts", maxAttempts)
}
