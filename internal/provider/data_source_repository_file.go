package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &repositoryFileDataSource{}
	_ datasource.DataSourceWithConfigure = &repositoryFileDataSource{}
)

func NewRepositoryFileDataSource() datasource.DataSource {
	return &repositoryFileDataSource{}
}

type repositoryFileDataSource struct {
	client *github.Client
	owner  string
}

type repositoryFileDataSourceModel struct {
	Repository    types.String `tfsdk:"repository"`
	FullName      types.String `tfsdk:"full_name"`
	File          types.String `tfsdk:"file"`
	Branch        types.String `tfsdk:"branch"`
	Ref           types.String `tfsdk:"ref"`
	Content       types.String `tfsdk:"content"`
	CommitSHA     types.String `tfsdk:"commit_sha"`
	CommitMessage types.String `tfsdk:"commit_message"`
	CommitAuthor  types.String `tfsdk:"commit_author"`
	CommitEmail   types.String `tfsdk:"commit_email"`
	SHA           types.String `tfsdk:"sha"`
	ID            types.String `tfsdk:"id"`
}

func (d *repositoryFileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_file"
}

func (d *repositoryFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a file in a GitHub repository.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository. Conflicts with `full_name`. If `repository` is provided, the provider-level `owner` configuration will be used.",
				Optional:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the repository (owner/repo). Conflicts with `repository`.",
				Optional:    true,
			},
			"file": schema.StringAttribute{
				Description: "The file path to read.",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The branch name, defaults to the repository's default branch.",
				Optional:    true,
			},
			"ref": schema.StringAttribute{
				Description: "The name of the commit/branch/tag.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "The file's content.",
				Computed:    true,
			},
			"commit_sha": schema.StringAttribute{
				Description: "The SHA of the commit that modified the file.",
				Computed:    true,
			},
			"commit_message": schema.StringAttribute{
				Description: "The commit message when the file was last modified.",
				Computed:    true,
			},
			"commit_author": schema.StringAttribute{
				Description: "The commit author name.",
				Computed:    true,
			},
			"commit_email": schema.StringAttribute{
				Description: "The commit author email address.",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "The blob SHA of the file.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Terraform state ID (repository/file).",
				Computed:    true,
			},
		},
	}
}

func (d *repositoryFileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *repositoryFileDataSource) getOwner(ctx context.Context) (string, error) {
	if d.owner != "" {
		return d.owner, nil
	}

	user, _, err := d.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and unable to fetch authenticated user: %v", err)
	}
	if user == nil || user.Login == nil {
		return "", fmt.Errorf("unable to determine owner: provider-level `owner` is not set and authenticated user information is unavailable")
	}
	return user.GetLogin(), nil
}

func (d *repositoryFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryFileDataSourceModel

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

	var owner, repoName string
	fullName := data.FullName.ValueString()
	repository := data.Repository.ValueString()

	if fullName != "" && repository != "" {
		resp.Diagnostics.AddError(
			"Conflicting Attributes",
			"Cannot specify both `full_name` and `repository`. Please use only one.",
		)
		return
	}

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

	filePath := data.File.ValueString()
	if filePath == "" {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The `file` attribute is required.",
		)
		return
	}

	opts := &github.RepositoryContentGetOptions{}
	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		opts.Ref = data.Branch.ValueString()
	}

	fc, dc, _, err := d.client.Repositories.GetContents(ctx, owner, repoName, filePath, opts)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[DEBUG] Missing GitHub repository file %s/%s/%s", owner, repoName, filePath)
				data.ID = types.StringValue("")
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error fetching GitHub repository file",
			fmt.Sprintf("Unable to fetch file %s from repository %s/%s: %v", filePath, owner, repoName, err),
		)
		return
	}

	data.Repository = types.StringValue(repoName)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", repoName, filePath))
	data.File = types.StringValue(filePath)

	if dc != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	content, err := fc.GetContent()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading file content",
			fmt.Sprintf("Unable to get content from file %s/%s/%s: %v", owner, repoName, filePath, err),
		)
		return
	}

	data.Content = types.StringValue(content)
	data.SHA = types.StringValue(fc.GetSHA())

	parsedURL, err := url.Parse(fc.GetURL())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error parsing file URL",
			fmt.Sprintf("Unable to parse file URL: %v", err),
		)
	} else {
		parsedQuery, err := url.ParseQuery(parsedURL.RawQuery)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error parsing query string",
				fmt.Sprintf("Unable to parse query string: %v", err),
			)
		} else {
			if refValues, ok := parsedQuery["ref"]; ok && len(refValues) > 0 {
				data.Ref = types.StringValue(refValues[0])
			} else {
				data.Ref = types.StringNull()
			}
		}
	}

	ref := data.Ref.ValueString()
	if ref == "" && !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		ref = data.Branch.ValueString()
	}

	if ref != "" {
		log.Printf("[DEBUG] Fetching commit info for repository file: %s/%s/%s", owner, repoName, filePath)
		commit, err := d.getFileCommit(ctx, owner, repoName, filePath, ref)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error fetching commit information",
				fmt.Sprintf("Unable to fetch commit information for file %s/%s/%s: %v", owner, repoName, filePath, err),
			)
		} else {
			log.Printf("[DEBUG] Found file: %s/%s/%s, in commit SHA: %s", owner, repoName, filePath, commit.GetSHA())
			data.CommitSHA = types.StringValue(commit.GetSHA())

			if commit.Commit != nil && commit.Commit.Committer != nil {
				data.CommitAuthor = types.StringValue(commit.Commit.Committer.GetName())
				data.CommitEmail = types.StringValue(commit.Commit.Committer.GetEmail())
			} else {
				data.CommitAuthor = types.StringNull()
				data.CommitEmail = types.StringNull()
			}

			if commit.Commit != nil {
				data.CommitMessage = types.StringValue(commit.Commit.GetMessage())
			} else {
				data.CommitMessage = types.StringNull()
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *repositoryFileDataSource) getFileCommit(ctx context.Context, owner, repo, file, ref string) (*github.RepositoryCommit, error) {
	opts := &github.CommitsListOptions{
		Path: file,
		SHA:  ref,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	commits, _, err := d.client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found for file %s in ref %s", file, ref)
	}

	commitSHA := commits[0].GetSHA()
	commit, _, err := d.client.Repositories.GetCommit(ctx, owner, repo, commitSHA, nil)
	if err != nil {
		return nil, err
	}

	return commit, nil
}
