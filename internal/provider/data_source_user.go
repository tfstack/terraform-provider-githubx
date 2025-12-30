package provider

import (
	"context"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *github.Client
}

// userDataSourceModel maps the data source schema data.
type userDataSourceModel struct {
	Username    types.String `tfsdk:"username"`
	ID          types.String `tfsdk:"id"`      // This is used as the Terraform state ID
	UserID      types.Int64  `tfsdk:"user_id"` // The GitHub user ID as integer
	NodeID      types.String `tfsdk:"node_id"`
	AvatarURL   types.String `tfsdk:"avatar_url"`
	HTMLURL     types.String `tfsdk:"html_url"`
	Name        types.String `tfsdk:"name"`
	Company     types.String `tfsdk:"company"`
	Blog        types.String `tfsdk:"blog"`
	Location    types.String `tfsdk:"location"`
	Email       types.String `tfsdk:"email"`
	Bio         types.String `tfsdk:"bio"`
	PublicRepos types.Int64  `tfsdk:"public_repos"`
	PublicGists types.Int64  `tfsdk:"public_gists"`
	Followers   types.Int64  `tfsdk:"followers"`
	Following   types.Int64  `tfsdk:"following"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Metadata returns the data source type name.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source.
func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub user.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "The GitHub username to look up.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The GitHub user ID (as string for Terraform state ID).",
				Computed:    true,
			},
			"user_id": schema.Int64Attribute{
				Description: "The GitHub user ID as an integer.",
				Computed:    true,
			},
			"node_id": schema.StringAttribute{
				Description: "The GitHub node ID of the user.",
				Computed:    true,
			},
			"avatar_url": schema.StringAttribute{
				Description: "The URL of the user's avatar.",
				Computed:    true,
			},
			"html_url": schema.StringAttribute{
				Description: "The GitHub URL of the user's profile.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The user's display name.",
				Computed:    true,
			},
			"company": schema.StringAttribute{
				Description: "The user's company.",
				Computed:    true,
			},
			"blog": schema.StringAttribute{
				Description: "The user's blog URL.",
				Computed:    true,
			},
			"location": schema.StringAttribute{
				Description: "The user's location.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user's email address.",
				Computed:    true,
			},
			"bio": schema.StringAttribute{
				Description: "The user's bio.",
				Computed:    true,
			},
			"public_repos": schema.Int64Attribute{
				Description: "The number of public repositories.",
				Computed:    true,
			},
			"public_gists": schema.Int64Attribute{
				Description: "The number of public gists.",
				Computed:    true,
			},
			"followers": schema.Int64Attribute{
				Description: "The number of followers.",
				Computed:    true,
			},
			"following": schema.Int64Attribute{
				Description: "The number of users following.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the user account was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the user account was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure enables provider-level data or clients to be set in the
// provider-defined data source type. It is separately executed for each
// ReadDataSource RPC.
func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
}

// Read refreshes the Terraform state with the latest data.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userDataSourceModel

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

	username := data.Username.ValueString()
	if username == "" {
		resp.Diagnostics.AddError(
			"Missing Username",
			"The username attribute is required.",
		)
		return
	}

	// Fetch the user from GitHub
	user, _, err := d.client.Users.Get(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching GitHub user",
			fmt.Sprintf("Unable to fetch user %s: %v", username, err),
		)
		return
	}

	// Map response body to model
	// Convert ID to string for Terraform state ID
	userID := user.GetID()
	data.ID = types.StringValue(fmt.Sprintf("%d", userID))
	data.UserID = types.Int64Value(userID)
	data.NodeID = types.StringValue(user.GetNodeID())
	data.AvatarURL = types.StringValue(user.GetAvatarURL())
	data.HTMLURL = types.StringValue(user.GetHTMLURL())
	data.Name = types.StringValue(user.GetName())
	data.Company = types.StringValue(user.GetCompany())
	data.Blog = types.StringValue(user.GetBlog())
	data.Location = types.StringValue(user.GetLocation())
	data.Email = types.StringValue(user.GetEmail())
	data.Bio = types.StringValue(user.GetBio())
	data.PublicRepos = types.Int64Value(int64(user.GetPublicRepos()))
	data.PublicGists = types.Int64Value(int64(user.GetPublicGists()))
	data.Followers = types.Int64Value(int64(user.GetFollowers()))
	data.Following = types.Int64Value(int64(user.GetFollowing()))

	if user.CreatedAt != nil {
		data.CreatedAt = types.StringValue(user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	if user.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Set ID for the data source (using username as the ID)
	data.Username = types.StringValue(username)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
