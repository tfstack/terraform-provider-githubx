package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &repositoryBranchResource{}
	_ resource.ResourceWithConfigure   = &repositoryBranchResource{}
	_ resource.ResourceWithImportState = &repositoryBranchResource{}
)

// NewRepositoryBranchResource is a helper function to simplify the provider implementation.
func NewRepositoryBranchResource() resource.Resource {
	return &repositoryBranchResource{}
}

// repositoryBranchResource is the resource implementation.
type repositoryBranchResource struct {
	client *github.Client
	owner  string
}

// repositoryBranchResourceModel maps the resource schema data.
type repositoryBranchResourceModel struct {
	Repository   types.String `tfsdk:"repository"`
	Branch       types.String `tfsdk:"branch"`
	SourceBranch types.String `tfsdk:"source_branch"`
	SourceSHA    types.String `tfsdk:"source_sha"`
	ETag         types.String `tfsdk:"etag"`
	Ref          types.String `tfsdk:"ref"`
	SHA          types.String `tfsdk:"sha"`
	ID           types.String `tfsdk:"id"`
}

// Metadata returns the resource type name.
func (r *repositoryBranchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_branch"
}

// Schema defines the schema for the resource.
func (r *repositoryBranchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a GitHub repository branch.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch": schema.StringAttribute{
				Description: "The repository branch to create.",
				Required:    true,
			},
			"source_branch": schema.StringAttribute{
				Description: "The branch name to start from. Defaults to 'main'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_sha": schema.StringAttribute{
				Description: "The commit hash to start from. Defaults to the tip of 'source_branch'. If provided, 'source_branch' is ignored.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the Branch object.",
				Optional:    true,
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "A string representing a branch reference, in the form of 'refs/heads/<branch>'.",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "A string storing the reference's HEAD commit's SHA1.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository/branch).",
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
func (r *repositoryBranchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

// Create creates the resource and sets the initial Terraform state.
func (r *repositoryBranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryBranchResourceModel

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

	repoName := plan.Repository.ValueString()
	branchName := plan.Branch.ValueString()
	branchRefName := "refs/heads/" + branchName
	sourceBranchName := plan.SourceBranch.ValueString()
	if sourceBranchName == "" {
		sourceBranchName = "main"
	}
	sourceBranchRefName := "refs/heads/" + sourceBranchName

	// Get source SHA
	var sourceBranchSHA string
	if !plan.SourceSHA.IsNull() && !plan.SourceSHA.IsUnknown() && plan.SourceSHA.ValueString() != "" {
		// Use provided source SHA
		sourceBranchSHA = plan.SourceSHA.ValueString()
	} else {
		// Get SHA from source branch
		ref, _, refErr := r.client.Git.GetRef(ctx, owner, repoName, sourceBranchRefName)
		if refErr != nil {
			resp.Diagnostics.AddError(
				"Error querying source branch",
				fmt.Sprintf("Unable to query GitHub branch reference %s/%s (%s): %v", owner, repoName, sourceBranchRefName, refErr),
			)
			return
		}
		if ref.Object != nil && ref.Object.SHA != nil {
			sourceBranchSHA = *ref.Object.SHA
			// Set source_sha in plan for state
			plan.SourceSHA = types.StringValue(sourceBranchSHA)
		} else {
			resp.Diagnostics.AddError(
				"Invalid source branch",
				fmt.Sprintf("Source branch %s does not have a valid SHA", sourceBranchName),
			)
			return
		}
	}

	// Create the branch
	_, _, createErr := r.client.Git.CreateRef(ctx, owner, repoName, &github.Reference{
		Ref:    &branchRefName,
		Object: &github.GitObject{SHA: &sourceBranchSHA},
	})
	// If the branch already exists, rather than erroring out just continue on to reading the branch
	// This avoids the case where a repo with gitignore_template and branch are being created at the same time crashing terraform
	if createErr != nil && !strings.HasSuffix(createErr.Error(), "422 Reference already exists []") {
		resp.Diagnostics.AddError(
			"Error creating branch",
			fmt.Sprintf("Unable to create GitHub branch reference %s/%s (%s): %v", owner, repoName, branchRefName, createErr),
		)
		return
	}

	// Set ID using colon delimiter (standard Terraform pattern)
	plan.ID = types.StringValue(buildTwoPartID(repoName, branchName))

	// Read the branch to get all computed values
	r.readBranch(ctx, owner, repoName, branchName, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *repositoryBranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryBranchResourceModel

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

	// Parse ID (format: repository:branch or repository/branch for backward compatibility)
	id := state.ID.ValueString()
	repoName, branchName, err := parseTwoPartID(id, "repository", "branch")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:branch' or 'repository/branch'. Error: %v", id, err),
		)
		return
	}

	// Read the branch
	r.readBranch(ctx, owner, repoName, branchName, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Migrate ID to new format if it was in old format
	// This ensures backward compatibility and updates state to new format
	if !strings.Contains(state.ID.ValueString(), ":") {
		// Old format detected, update to new format
		state.ID = types.StringValue(buildTwoPartID(repoName, branchName))
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *repositoryBranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state repositoryBranchResourceModel

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

	// Parse ID from state (format: repository:branch or repository/branch for backward compatibility)
	id := state.ID.ValueString()
	repoName, oldBranchName, err := parseTwoPartID(id, "repository", "branch")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:branch' or 'repository/branch'. Error: %v", id, err),
		)
		return
	}
	newBranchName := plan.Branch.ValueString()

	// Check if branch name changed
	if !plan.Branch.Equal(state.Branch) {
		// Rename the branch
		_, _, err := r.client.Repositories.RenameBranch(ctx, owner, repoName, oldBranchName, newBranchName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error renaming branch",
				fmt.Sprintf("Unable to rename GitHub branch %s/%s (%s -> %s): %v", owner, repoName, oldBranchName, newBranchName, err),
			)
			return
		}

		// Update ID
		plan.ID = types.StringValue(buildTwoPartID(repoName, newBranchName))
	} else {
		plan.ID = state.ID
	}

	// Read the branch to get all computed values
	r.readBranch(ctx, owner, repoName, newBranchName, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *repositoryBranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryBranchResourceModel

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

	// Parse ID (format: repository:branch or repository/branch for backward compatibility)
	id := state.ID.ValueString()
	repoName, branchName, parseErr := parseTwoPartID(id, "repository", "branch")
	if parseErr != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:branch' or 'repository/branch'. Error: %v", id, parseErr),
		)
		return
	}
	branchRefName := "refs/heads/" + branchName

	// Delete the branch
	log.Printf("[DEBUG] Deleting branch: %s/%s (%s)", owner, repoName, branchRefName)
	_, err = r.client.Git.DeleteRef(ctx, owner, repoName, branchRefName)
	if err != nil {
		// Check if the branch already doesn't exist (404 or 422 with "Reference does not exist")
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil {
			statusCode := ghErr.Response.StatusCode
			// 404: Branch not found
			// 422: Reference does not exist (branch was already deleted, e.g., by auto-merge)
			if statusCode == http.StatusNotFound ||
				(statusCode == http.StatusUnprocessableEntity &&
					strings.Contains(err.Error(), "Reference does not exist")) {
				log.Printf("[INFO] Branch %s/%s (%s) no longer exists, removing from state", owner, repoName, branchRefName)
				return // Successfully removed from state
			}
		}
		resp.Diagnostics.AddError(
			"Error deleting branch",
			fmt.Sprintf("Unable to delete GitHub branch reference %s/%s (%s): %v", owner, repoName, branchRefName, err),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *repositoryBranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (format: repository:branch or repository:branch:source_branch)
	// Use colon as delimiter (standard Terraform pattern)
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) < 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'repository:branch' or 'repository:branch:source_branch'.",
		)
		return
	}

	repoName := parts[0]
	branchName := parts[1]

	// Check if source_branch is specified (third part)
	var sourceBranch string
	if len(parts) == 3 {
		sourceBranch = parts[2]
	} else {
		sourceBranch = "main"
	}

	// Set the ID using colon delimiter
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), buildTwoPartID(repoName, branchName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), repoName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch"), branchName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_branch"), sourceBranch)...)
}

// Helper methods

// getOwner gets the owner, falling back to authenticated user if not set.
func (r *repositoryBranchResource) getOwner(ctx context.Context) (string, error) {
	if r.owner != "" {
		return r.owner, nil
	}
	// Try to get authenticated user
	user, _, err := r.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and unable to fetch authenticated user: %v", err)
	}
	if user == nil || user.Login == nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and authenticated user information is unavailable")
	}
	return user.GetLogin(), nil
}

// buildTwoPartID creates a two-part ID using colon as delimiter (standard Terraform pattern).
// Format: "repository:branch".
func buildTwoPartID(part1, part2 string) string {
	return fmt.Sprintf("%s:%s", part1, part2)
}

// parseTwoPartID parses a two-part ID using colon as delimiter (preferred) or slash (backward compatibility).
// Returns the two parts and an error if the format is invalid.
// Supports both "repository:branch" (new format) and "repository/branch" (old format) for backward compatibility.
func parseTwoPartID(id, part1Name, part2Name string) (string, string, error) {
	// Try colon delimiter first (new format)
	if strings.Contains(id, ":") {
		parts := strings.SplitN(id, ":", 2)
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
	}

	// Fall back to slash delimiter (old format for backward compatibility)
	// Use SplitN to only split on the first "/" since branch names can contain slashes
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected format of ID (%s), expected %s:%s or %s/%s", id, part1Name, part2Name, part1Name, part2Name)
}

// readBranch reads branch data from GitHub and populates the model.
func (r *repositoryBranchResource) readBranch(ctx context.Context, owner, repoName, branchName string, model *repositoryBranchResourceModel, diags *diag.Diagnostics) {
	branchRefName := "refs/heads/" + branchName

	ref, resp, err := r.client.Git.GetRef(ctx, owner, repoName, branchRefName)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if resp != nil && resp.StatusCode == http.StatusNotModified {
				// Branch hasn't changed, use existing state
				return
			}
			if (resp != nil && resp.StatusCode == http.StatusNotFound) ||
				(ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound) {
				log.Printf("[INFO] Removing branch %s/%s (%s) from state because it no longer exists in GitHub",
					owner, repoName, branchName)
				model.ID = types.StringValue("")
				return
			}
		}
		diags.AddError(
			"Error reading branch",
			fmt.Sprintf("Unable to read branch %s/%s (%s): %v", owner, repoName, branchRefName, err),
		)
		return
	}

	// Set ID using colon delimiter
	model.ID = types.StringValue(buildTwoPartID(repoName, branchName))

	// Set repository and branch
	model.Repository = types.StringValue(repoName)
	model.Branch = types.StringValue(branchName)

	// Set ETag from response header
	if resp != nil {
		model.ETag = types.StringValue(resp.Header.Get("ETag"))
	} else {
		model.ETag = types.StringNull()
	}

	// Set ref and SHA
	if ref != nil {
		if ref.Ref != nil {
			model.Ref = types.StringValue(*ref.Ref)
		} else {
			model.Ref = types.StringNull()
		}

		if ref.Object != nil && ref.Object.SHA != nil {
			model.SHA = types.StringValue(*ref.Object.SHA)
		} else {
			model.SHA = types.StringNull()
		}
	} else {
		model.Ref = types.StringNull()
		model.SHA = types.StringNull()
	}
}
