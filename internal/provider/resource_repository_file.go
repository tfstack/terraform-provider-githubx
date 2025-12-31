package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

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

var (
	_ resource.Resource                = &repositoryFileResource{}
	_ resource.ResourceWithConfigure   = &repositoryFileResource{}
	_ resource.ResourceWithImportState = &repositoryFileResource{}
)

func NewRepositoryFileResource() resource.Resource {
	return &repositoryFileResource{}
}

type repositoryFileResource struct {
	client *github.Client
	owner  string
}

type repositoryFileResourceModel struct {
	Repository                types.String `tfsdk:"repository"`
	File                      types.String `tfsdk:"file"`
	Content                   types.String `tfsdk:"content"`
	Branch                    types.String `tfsdk:"branch"`
	Ref                       types.String `tfsdk:"ref"`
	CommitSHA                 types.String `tfsdk:"commit_sha"`
	CommitMessage             types.String `tfsdk:"commit_message"`
	CommitAuthor              types.String `tfsdk:"commit_author"`
	CommitEmail               types.String `tfsdk:"commit_email"`
	SHA                       types.String `tfsdk:"sha"`
	OverwriteOnCreate         types.Bool   `tfsdk:"overwrite_on_create"`
	AutocreateBranch          types.Bool   `tfsdk:"autocreate_branch"`
	AutocreateBranchSource    types.String `tfsdk:"autocreate_branch_source_branch"`
	AutocreateBranchSourceSHA types.String `tfsdk:"autocreate_branch_source_sha"`
	ID                        types.String `tfsdk:"id"`
}

func (r *repositoryFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_file"
}

func (r *repositoryFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a file in a GitHub repository.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file": schema.StringAttribute{
				Description: "The file path to manage.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "The file's content.",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The branch name, defaults to the repository's default branch.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ref": schema.StringAttribute{
				Description: "The name of the commit/branch/tag.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"commit_sha": schema.StringAttribute{
				Description: "The SHA of the commit that modified the file.",
				Computed:    true,
			},
			"commit_message": schema.StringAttribute{
				Description: "The commit message when creating, updating or deleting the file.",
				Optional:    true,
				Computed:    true,
			},
			"commit_author": schema.StringAttribute{
				Description: "The commit author name, defaults to the authenticated user's name. GitHub app users may omit author and email information so GitHub can verify commits as the GitHub App.",
				Optional:    true,
			},
			"commit_email": schema.StringAttribute{
				Description: "The commit author email address, defaults to the authenticated user's email address. GitHub app users may omit author and email information so GitHub can verify commits as the GitHub App.",
				Optional:    true,
			},
			"sha": schema.StringAttribute{
				Description: "The blob SHA of the file.",
				Computed:    true,
			},
			"overwrite_on_create": schema.BoolAttribute{
				Description: "Enable overwriting existing files, defaults to \"false\".",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"autocreate_branch": schema.BoolAttribute{
				Description: "Automatically create the branch if it could not be found. Subsequent reads if the branch is deleted will occur from 'autocreate_branch_source_branch'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"autocreate_branch_source_branch": schema.StringAttribute{
				Description: "The branch name to start from, if 'autocreate_branch' is set. Defaults to 'main'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
			},
			"autocreate_branch_source_sha": schema.StringAttribute{
				Description: "The commit hash to start from, if 'autocreate_branch' is set. Defaults to the tip of 'autocreate_branch_source_branch'. If provided, 'autocreate_branch_source_branch' is ignored.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository:file).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *repositoryFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *repositoryFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryFileResourceModel

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
	filePath := plan.File.ValueString()
	content := plan.Content.ValueString()

	if !r.checkAndCreateBranchIfNeeded(ctx, owner, repoName, &plan, &resp.Diagnostics) {
		return
	}

	opts, diags := r.buildFileOptions(ctx, &plan, content)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if opts.Message == nil || *opts.Message == "" {
		msg := fmt.Sprintf("Add %s", filePath)
		opts.Message = &msg
		plan.CommitMessage = types.StringValue(msg)
	}

	if plan.OverwriteOnCreate.ValueBool() {
		checkOpts := &github.RepositoryContentGetOptions{}
		if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
			checkOpts.Ref = plan.Branch.ValueString()
		}
		fc, _, _, err := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, checkOpts)
		if err == nil && fc != nil {
			opts.SHA = github.String(fc.GetSHA())
		}
	}

	var create *github.RepositoryContentResponse
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		create, _, err = r.client.Repositories.CreateFile(ctx, owner, repoName, filePath, opts)
		if err == nil {
			break
		}
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusConflict {
			checkOpts := &github.RepositoryContentGetOptions{}
			if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
				checkOpts.Ref = plan.Branch.ValueString()
			}
			fc, _, _, readErr := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, checkOpts)
			if readErr == nil && fc != nil {
				if plan.OverwriteOnCreate.ValueBool() {
					opts.SHA = github.String(fc.GetSHA())
					continue
				} else {
					resp.Diagnostics.AddError(
						"File Already Exists",
						fmt.Sprintf("File %s already exists in repository %s/%s. Set 'overwrite_on_create' to true to overwrite it.", filePath, owner, repoName),
					)
					return
				}
			}
			continue
		}
		break
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating file",
			fmt.Sprintf("Unable to create file %s in repository %s/%s after %d attempts: %v", filePath, owner, repoName, maxRetries, err),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, filePath))
	if create != nil {
		plan.CommitSHA = types.StringValue(create.GetSHA())
	}

	r.readFile(ctx, owner, repoName, filePath, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.AutocreateBranch.ValueBool() {
		plan.AutocreateBranchSourceSHA = types.StringNull()
	} else if plan.AutocreateBranchSourceSHA.IsNull() || plan.AutocreateBranchSourceSHA.IsUnknown() {
		plan.AutocreateBranchSourceSHA = types.StringNull()
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repositoryFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryFileResourceModel

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
	if id == "" {
		// ID is empty, resource should be removed from state
		log.Printf("[INFO] Resource ID is empty, removing from state")
		state.ID = types.StringValue("")
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:file'.", id),
		)
		return
	}

	repoName := parts[0]
	filePath := parts[1]

	if !state.Branch.IsNull() && !state.Branch.IsUnknown() {
		branchName := state.Branch.ValueString()
		if err := r.checkRepositoryBranchExists(ctx, owner, repoName, branchName); err != nil {
			if state.AutocreateBranch.ValueBool() {
				state.Branch = state.AutocreateBranchSource
			} else {
				log.Printf("[INFO] Removing repository file %s/%s/%s from state because the branch no longer exists in GitHub",
					owner, repoName, filePath)
				state.ID = types.StringValue("")
				resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
				return
			}
		}
	}

	r.readFile(ctx, owner, repoName, filePath, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.AutocreateBranch.ValueBool() {
		state.AutocreateBranchSourceSHA = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *repositoryFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state repositoryFileResourceModel

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
	if id == "" {
		// ID is empty, resource should be removed from state
		log.Printf("[INFO] Resource ID is empty during update, removing from state")
		state.ID = types.StringValue("")
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:file'.", id),
		)
		return
	}

	repoName := parts[0]
	filePath := parts[1]
	content := plan.Content.ValueString()

	if !r.checkAndCreateBranchIfNeeded(ctx, owner, repoName, &plan, &resp.Diagnostics) {
		return
	}

	opts, diags := r.buildFileOptions(ctx, &plan, content)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if !state.SHA.IsNull() && !state.SHA.IsUnknown() {
		opts.SHA = github.String(state.SHA.ValueString())
	}

	if opts.Message == nil || *opts.Message == "" || *opts.Message == fmt.Sprintf("Add %s", filePath) {
		msg := fmt.Sprintf("Update %s", filePath)
		opts.Message = &msg
		plan.CommitMessage = types.StringValue(msg)
	}

	var create *github.RepositoryContentResponse
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		create, _, err = r.client.Repositories.CreateFile(ctx, owner, repoName, filePath, opts)
		if err == nil {
			break
		}
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusConflict {
			updateOpts := &github.RepositoryContentGetOptions{}
			if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
				updateOpts.Ref = plan.Branch.ValueString()
			}
			fc, _, _, retryErr := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, updateOpts)
			if retryErr == nil && fc != nil {
				opts.SHA = github.String(fc.GetSHA())
				continue
			}
		}
		break
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating file",
			fmt.Sprintf("Unable to update file %s in repository %s/%s after %d attempts: %v", filePath, owner, repoName, maxRetries, err),
		)
		return
	}

	plan.ID = state.ID
	plan.CommitSHA = types.StringValue(create.GetSHA())

	r.readFile(ctx, owner, repoName, filePath, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repositoryFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryFileResourceModel

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
	if id == "" {
		// ID is empty, resource already removed from state
		log.Printf("[INFO] Resource ID is empty during delete, nothing to delete")
		return
	}

	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Invalid ID format: %s. Expected 'repository:file'.", id),
		)
		return
	}

	repoName := parts[0]
	filePath := parts[1]

	if !r.checkAndCreateBranchIfNeeded(ctx, owner, repoName, &state, &resp.Diagnostics) {
		return
	}

	opts, diags := r.buildFileOptions(ctx, &state, "")
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if opts.Message == nil || *opts.Message == "" || *opts.Message == fmt.Sprintf("Add %s", filePath) {
		msg := fmt.Sprintf("Delete %s", filePath)
		opts.Message = &msg
	}

	if !state.SHA.IsNull() && !state.SHA.IsUnknown() {
		opts.SHA = github.String(state.SHA.ValueString())
	}

	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, _, err = r.client.Repositories.DeleteFile(ctx, owner, repoName, filePath, opts)
		if err == nil {
			return
		}
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response != nil {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				return
			} else if ghErr.Response.StatusCode == http.StatusConflict {
				getOpts := &github.RepositoryContentGetOptions{}
				if !state.Branch.IsNull() && !state.Branch.IsUnknown() {
					getOpts.Ref = state.Branch.ValueString()
				}
				fc, _, _, readErr := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, getOpts)
				if readErr != nil {
					var readGhErr *github.ErrorResponse
					if errors.As(readErr, &readGhErr) && readGhErr.Response != nil && readGhErr.Response.StatusCode == http.StatusNotFound {
						return
					}
				}
				if fc != nil {
					opts.SHA = github.String(fc.GetSHA())
					continue
				}
			}
		}
		break
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting file",
			fmt.Sprintf("Unable to delete file %s from repository %s/%s after %d attempts: %v", filePath, owner, repoName, maxRetries, err),
		)
		return
	}
}

func (r *repositoryFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) < 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'repository:file' or 'repository:file:branch'.",
		)
		return
	}

	repoName := parts[0]
	filePath := parts[1]
	var branch string
	if len(parts) == 3 {
		branch = parts[2]
	}

	owner, err := r.getOwner(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing Owner",
			fmt.Sprintf("Unable to determine owner: %v", err),
		)
		return
	}

	opts := &github.RepositoryContentGetOptions{}
	if branch != "" {
		opts.Ref = branch
	}

	fc, _, _, err := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing file",
			fmt.Sprintf("Unable to read file %s from repository %s/%s: %v", filePath, owner, repoName, err),
		)
		return
	}

	if fc == nil {
		resp.Diagnostics.AddError(
			"File Not Found",
			fmt.Sprintf("File %s is not a file in repository %s/%s or repository is not readable", filePath, owner, repoName),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%s", repoName, filePath))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), repoName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file"), filePath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("overwrite_on_create"), false)...)
	if branch != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch"), branch)...)
	}
}

func (r *repositoryFileResource) getOwner(ctx context.Context) (string, error) {
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

func (r *repositoryFileResource) checkRepositoryBranchExists(ctx context.Context, owner, repo, branch string) error {
	branchRefName := "refs/heads/" + branch
	_, _, err := r.client.Git.GetRef(ctx, owner, repo, branchRefName)
	return err
}

func (r *repositoryFileResource) checkAndCreateBranchIfNeeded(ctx context.Context, owner, repo string, model *repositoryFileResourceModel, diags *diag.Diagnostics) bool {
	if model.Branch.IsNull() || model.Branch.IsUnknown() {
		return true
	}

	branchName := model.Branch.ValueString()
	if err := r.checkRepositoryBranchExists(ctx, owner, repo, branchName); err != nil {
		if !model.AutocreateBranch.ValueBool() {
			diags.AddError(
				"Branch Not Found",
				fmt.Sprintf("Branch %s not found in repository %s/%s. Set 'autocreate_branch' to true to automatically create it.", branchName, owner, repo),
			)
			return false
		}

		branchRefName := "refs/heads/" + branchName
		sourceBranchName := model.AutocreateBranchSource.ValueString()
		if sourceBranchName == "" {
			sourceBranchName = "main"
		}
		sourceBranchRefName := "refs/heads/" + sourceBranchName

		var sourceBranchSHA string
		if !model.AutocreateBranchSourceSHA.IsNull() && !model.AutocreateBranchSourceSHA.IsUnknown() && model.AutocreateBranchSourceSHA.ValueString() != "" {
			sourceBranchSHA = model.AutocreateBranchSourceSHA.ValueString()
		} else {
			ref, _, err := r.client.Git.GetRef(ctx, owner, repo, sourceBranchRefName)
			if err != nil {
				diags.AddError(
					"Error querying source branch",
					fmt.Sprintf("Unable to query GitHub branch reference %s/%s (%s): %v", owner, repo, sourceBranchRefName, err),
				)
				return false
			}
			if ref.Object != nil && ref.Object.SHA != nil {
				sourceBranchSHA = *ref.Object.SHA
				model.AutocreateBranchSourceSHA = types.StringValue(sourceBranchSHA)
			} else {
				diags.AddError(
					"Invalid source branch",
					fmt.Sprintf("Source branch %s does not have a valid SHA", sourceBranchName),
				)
				return false
			}
		}

		_, _, err := r.client.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref:    &branchRefName,
			Object: &github.GitObject{SHA: &sourceBranchSHA},
		})
		if err != nil {
			diags.AddError(
				"Error creating branch",
				fmt.Sprintf("Unable to create GitHub branch reference %s/%s (%s): %v", owner, repo, branchRefName, err),
			)
			return false
		}
	}
	return true
}

func (r *repositoryFileResource) buildFileOptions(_ context.Context, model *repositoryFileResourceModel, content string) (*github.RepositoryContentFileOptions, diag.Diagnostics) {
	var diags diag.Diagnostics

	opts := &github.RepositoryContentFileOptions{
		Content: []byte(content),
	}

	if !model.Branch.IsNull() && !model.Branch.IsUnknown() {
		opts.Branch = github.String(model.Branch.ValueString())
	}

	if !model.CommitMessage.IsNull() && !model.CommitMessage.IsUnknown() {
		msg := model.CommitMessage.ValueString()
		opts.Message = &msg
	}

	hasCommitAuthor := !model.CommitAuthor.IsNull() && !model.CommitAuthor.IsUnknown()
	hasCommitEmail := !model.CommitEmail.IsNull() && !model.CommitEmail.IsUnknown()

	if hasCommitAuthor && !hasCommitEmail {
		diags.AddError(
			"Invalid Commit Author Configuration",
			"Cannot set commit_author without setting commit_email",
		)
		return nil, diags
	}

	if hasCommitEmail && !hasCommitAuthor {
		diags.AddError(
			"Invalid Commit Author Configuration",
			"Cannot set commit_email without setting commit_author",
		)
		return nil, diags
	}

	if hasCommitAuthor && hasCommitEmail {
		name := model.CommitAuthor.ValueString()
		email := model.CommitEmail.ValueString()
		opts.Author = &github.CommitAuthor{Name: &name, Email: &email}
		opts.Committer = &github.CommitAuthor{Name: &name, Email: &email}
	}

	return opts, diags
}

func (r *repositoryFileResource) readFile(ctx context.Context, owner, repoName, filePath string, model *repositoryFileResourceModel, diags *diag.Diagnostics) {
	opts := &github.RepositoryContentGetOptions{}

	if !model.Branch.IsNull() && !model.Branch.IsUnknown() {
		opts.Ref = model.Branch.ValueString()
	}

	fc, _, _, err := r.client.Repositories.GetContents(ctx, owner, repoName, filePath, opts)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusTooManyRequests {
				diags.AddError(
					"Rate Limit Exceeded",
					fmt.Sprintf("GitHub API rate limit exceeded: %v", err),
				)
				return
			}
			if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing repository file %s/%s/%s from state because it no longer exists in GitHub",
					owner, repoName, filePath)
				model.ID = types.StringValue("")
				return
			}
		}
		diags.AddError(
			"Error reading file",
			fmt.Sprintf("Unable to read file %s from repository %s/%s: %v", filePath, owner, repoName, err),
		)
		return
	}

	if fc == nil {
		log.Printf("[INFO] Removing repository file %s/%s/%s from state because it no longer exists in GitHub",
			owner, repoName, filePath)
		model.ID = types.StringValue("")
		return
	}

	content, err := fc.GetContent()
	if err != nil {
		diags.AddError(
			"Error reading file content",
			fmt.Sprintf("Unable to get content from file %s/%s/%s: %v", owner, repoName, filePath, err),
		)
		return
	}

	model.Content = types.StringValue(content)
	model.Repository = types.StringValue(repoName)
	model.File = types.StringValue(filePath)
	model.SHA = types.StringValue(fc.GetSHA())

	parsedURL, err := url.Parse(fc.GetURL())
	if err != nil {
		diags.AddWarning(
			"Error parsing file URL",
			fmt.Sprintf("Unable to parse file URL: %v", err),
		)
	} else {
		parsedQuery, err := url.ParseQuery(parsedURL.RawQuery)
		if err != nil {
			diags.AddWarning(
				"Error parsing query string",
				fmt.Sprintf("Unable to parse query string: %v", err),
			)
		} else {
			if refValues, ok := parsedQuery["ref"]; ok && len(refValues) > 0 {
				model.Ref = types.StringValue(refValues[0])
			} else {
				model.Ref = types.StringNull()
			}
		}
	}

	ref := model.Ref.ValueString()
	if ref == "" && !model.Branch.IsNull() && !model.Branch.IsUnknown() {
		ref = model.Branch.ValueString()
	}

	if ref != "" {
		var commit *github.RepositoryCommit
		if !model.CommitSHA.IsNull() && !model.CommitSHA.IsUnknown() {
			commit, _, err = r.client.Repositories.GetCommit(ctx, owner, repoName, model.CommitSHA.ValueString(), nil)
		} else {
			commit, err = r.getFileCommit(ctx, owner, repoName, filePath, ref)
		}
		if err != nil {
			diags.AddWarning(
				"Error fetching commit information",
				fmt.Sprintf("Unable to fetch commit information for file %s/%s/%s: %v", owner, repoName, filePath, err),
			)
		} else {
			model.CommitSHA = types.StringValue(commit.GetSHA())

			if commit.Commit != nil {
				// Only set commit message if not already set to preserve user-provided value
				// (GitHub may normalize trailing newlines)
				if model.CommitMessage.IsNull() || model.CommitMessage.IsUnknown() {
					model.CommitMessage = types.StringValue(commit.Commit.GetMessage())
				}

				if commit.Commit.Committer != nil {
					commitAuthor := commit.Commit.Committer.GetName()
					commitEmail := commit.Commit.Committer.GetEmail()

					hasCommitAuthor := !model.CommitAuthor.IsNull() && !model.CommitAuthor.IsUnknown()
					hasCommitEmail := !model.CommitEmail.IsNull() && !model.CommitEmail.IsUnknown()

					if commitAuthor != "GitHub" && commitEmail != "noreply@github.com" && hasCommitAuthor && hasCommitEmail {
						model.CommitAuthor = types.StringValue(commitAuthor)
						model.CommitEmail = types.StringValue(commitEmail)
					}
				}
			}
		}
	}
}

func (r *repositoryFileResource) getFileCommit(ctx context.Context, owner, repo, file, ref string) (*github.RepositoryCommit, error) {
	opts := &github.CommitsListOptions{
		Path: file,
		SHA:  ref,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	commits, _, err := r.client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found for file %s in ref %s", file, ref)
	}

	commitSHA := commits[0].GetSHA()
	commit, _, err := r.client.Repositories.GetCommit(ctx, owner, repo, commitSHA, nil)
	if err != nil {
		return nil, err
	}

	return commit, nil
}
