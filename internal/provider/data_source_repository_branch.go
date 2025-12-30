package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &repositoryBranchDataSource{}
	_ datasource.DataSourceWithConfigure = &repositoryBranchDataSource{}
)

// NewRepositoryBranchDataSource is a helper function to simplify the provider implementation.
func NewRepositoryBranchDataSource() datasource.DataSource {
	return &repositoryBranchDataSource{}
}

// repositoryBranchDataSource is the data source implementation.
type repositoryBranchDataSource struct {
	client *github.Client
	owner  string
}

// repositoryBranchDataSourceModel maps the data source schema data.
type repositoryBranchDataSourceModel struct {
	Repository types.String `tfsdk:"repository"`
	FullName   types.String `tfsdk:"full_name"`
	Branch     types.String `tfsdk:"branch"`
	ETag       types.String `tfsdk:"etag"`
	Ref        types.String `tfsdk:"ref"`
	SHA        types.String `tfsdk:"sha"`
	ID         types.String `tfsdk:"id"`
}

// Metadata returns the data source type name.
func (d *repositoryBranchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_branch"
}

// Schema defines the schema for the data source.
func (d *repositoryBranchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub repository branch.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository. Conflicts with `full_name`. If `repository` is provided, the provider-level `owner` configuration will be used.",
				Optional:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the repository (owner/repo). Conflicts with `repository`.",
				Optional:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The name of the branch.",
				Required:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag header value from the API response.",
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "The full Git reference (e.g., refs/heads/main).",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "The SHA of the commit that the branch points to.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository/branch).",
				Computed:    true,
			},
		},
	}
}

// Configure enables provider-level data or clients to be set in the
// provider-defined data source type. It is separately executed for each
// ReadDataSource RPC.
func (d *repositoryBranchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *repositoryBranchDataSource) getOwner(ctx context.Context) (string, error) {
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
func (d *repositoryBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryBranchDataSourceModel

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
	repository := data.Repository.ValueString()

	// Check for conflicts
	if fullName != "" && repository != "" {
		resp.Diagnostics.AddError(
			"Conflicting Attributes",
			"Cannot specify both `full_name` and `repository`. Please use only one.",
		)
		return
	}

	// Parse full_name or use repository with owner
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
	} else if repository != "" {
		repoName = repository
		var err error
		owner, err = d.getOwner(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Missing Owner",
				fmt.Sprintf("Either `full_name` must be provided, or `repository` must be provided along with provider-level `owner` configuration or authentication. Error: %v", err),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either `full_name` or `repository` must be provided.",
		)
		return
	}

	// Get branch name
	branchName := data.Branch.ValueString()
	if branchName == "" {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The `branch` attribute is required.",
		)
		return
	}

	// Build the Git reference name
	branchRefName := "refs/heads/" + branchName

	// Fetch the branch reference from GitHub
	ref, ghResp, err := d.client.Git.GetRef(ctx, owner, repoName, branchRefName)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			// Check both ghResp and ghErr.Response for 404 status
			if (ghResp != nil && ghResp.StatusCode == http.StatusNotFound) ||
				(ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound) {
				log.Printf("[DEBUG] Missing GitHub branch %s/%s (%s)", owner, repoName, branchRefName)
				resp.Diagnostics.AddWarning(
					"Branch Not Found",
					fmt.Sprintf("Branch %s not found in repository %s/%s. Setting empty state.", branchName, owner, repoName),
				)
				data.ID = types.StringValue("")
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error fetching GitHub branch",
			fmt.Sprintf("Unable to fetch branch %s from repository %s/%s: %v", branchName, owner, repoName, err),
		)
		return
	}

	// Set the ID (repository/branch format)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", repoName, branchName))

	// Set ETag from response header
	if ghResp != nil {
		data.ETag = types.StringValue(ghResp.Header.Get("ETag"))
	} else {
		data.ETag = types.StringNull()
	}

	// Set ref and SHA
	if ref != nil {
		if ref.Ref != nil {
			data.Ref = types.StringValue(*ref.Ref)
		} else {
			data.Ref = types.StringNull()
		}

		if ref.Object != nil && ref.Object.SHA != nil {
			data.SHA = types.StringValue(*ref.Object.SHA)
		} else {
			data.SHA = types.StringNull()
		}
	} else {
		data.Ref = types.StringNull()
		data.SHA = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
