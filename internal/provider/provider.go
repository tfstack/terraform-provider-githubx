package provider

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &githubxProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &githubxProvider{
			version: version,
		}
	}
}

// githubxProvider is the provider implementation.
type githubxProvider struct {
	version string
}

// githubxProviderModel maps provider schema data to a Go type.
type githubxProviderModel struct {
	Token      types.String  `tfsdk:"token"`
	OAuthToken types.String  `tfsdk:"oauth_token"`
	AppAuth    *appAuthModel `tfsdk:"app_auth"`
	BaseURL    types.String  `tfsdk:"base_url"`
	Owner      types.String  `tfsdk:"owner"`
	Insecure   types.Bool    `tfsdk:"insecure"`
}

// appAuthModel represents GitHub App authentication configuration.
type appAuthModel struct {
	ID             types.Int64  `tfsdk:"id"`
	InstallationID types.Int64  `tfsdk:"installation_id"`
	PEMFile        types.String `tfsdk:"pem_file"`
}

// githubxClientData holds the GitHub API client for use in resources and data sources.
type githubxClientData struct {
	Client *github.Client
	Owner  string
}

// Metadata returns the provider type name.
func (p *githubxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "githubx"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *githubxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "GitHub personal access token for authentication. This token is required to authenticate with the GitHub API. You can obtain a token from GitHub Settings > Developer settings > Personal access tokens. Alternatively, you can set the GITHUB_TOKEN environment variable, or the provider will automatically use GitHub CLI authentication (gh auth token) if available.",
			},
			"oauth_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "GitHub OAuth token for authentication. This is an alternative to the personal access token.",
			},
			"app_auth": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GitHub App authentication configuration. Requires app_id, installation_id, and pem_file.",
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Required:    true,
						Description: "The GitHub App ID.",
					},
					"installation_id": schema.Int64Attribute{
						Required:    true,
						Description: "The GitHub App installation ID.",
					},
					"pem_file": schema.StringAttribute{
						Required:    true,
						Sensitive:   false,
						Description: "Path to the GitHub App private key PEM file.",
					},
				},
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The GitHub Base API URL. Defaults to `https://api.github.com/`. Set this to your GitHub Enterprise Server API URL (e.g., `https://github.example.com/api/v3/`).",
			},
			"owner": schema.StringAttribute{
				Optional:    true,
				Description: "The GitHub owner name to manage. Use this field when managing individual accounts or organizations.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable insecure mode for testing purposes. This disables TLS certificate verification. Use only in development/testing environments.",
			},
		},
	}
}

// Configure prepares a GitHub API client for data sources and resources.
func (p *githubxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config githubxProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Authentication precedence:
	// 1. Provider token attribute
	// 2. Provider oauth_token attribute
	// 3. GITHUB_TOKEN environment variable
	// 4. GitHub CLI (gh auth token)
	// 5. GitHub App authentication
	// 6. Unauthenticated (fallback)

	var client *github.Client
	var token string

	// 1. Check provider token attribute
	token = config.Token.ValueString()

	// 2. Check provider oauth_token attribute
	if token == "" {
		token = config.OAuthToken.ValueString()
	}

	// 3. Check GITHUB_TOKEN environment variable
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	// 4. Check GitHub CLI authentication
	if token == "" {
		if ghToken := getGitHubCLIToken(); ghToken != "" {
			token = ghToken
		}
	}

	// Parse base URL (default to GitHub.com API) - needed before App Auth
	baseURLStr := config.BaseURL.ValueString()
	if baseURLStr == "" {
		baseURLStr = os.Getenv("GITHUB_BASE_URL")
	}
	if baseURLStr == "" {
		baseURLStr = "https://api.github.com/"
	}

	// Parse and validate base URL
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Base URL",
			fmt.Sprintf("Unable to parse base_url: %v", err),
		)
		return
	}
	// Ensure base URL ends with /
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	// 5. Check GitHub App authentication (needs baseURL)
	if token == "" && config.AppAuth != nil {
		appToken, appResp := getGitHubAppToken(ctx, config.AppAuth, baseURL)
		if appResp != nil && appResp.Diagnostics.HasError() {
			resp.Diagnostics.Append(appResp.Diagnostics...)
			return
		}
		if appToken != "" {
			token = appToken
		}
	}

	// Get insecure mode setting
	insecure := config.Insecure.ValueBool()
	if !insecure {
		// Check environment variable
		if os.Getenv("GITHUB_INSECURE") == "true" {
			insecure = true
		}
	}

	// Create HTTP client with optional insecure TLS
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	} else {
		httpClient = &http.Client{}
	}

	// Configure insecure TLS if requested
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = tr
	}

	// Create GitHub client
	client = github.NewClient(httpClient)

	// Set base URL if not default
	if baseURL.String() != "https://api.github.com/" {
		client.BaseURL = baseURL
	}

	// Get owner from config or environment
	owner := config.Owner.ValueString()
	if owner == "" {
		owner = os.Getenv("GITHUB_OWNER")
	}

	// If owner is still not set and we have a token, fetch the authenticated user
	if owner == "" && token != "" {
		user, _, err := client.Users.Get(ctx, "")
		if err == nil && user != nil {
			owner = user.GetLogin()
		}
		// If we can't get the user, owner will remain empty and resources can try again
	}

	// Store the client for use in resources and data sources
	clientData := githubxClientData{
		Client: client,
		Owner:  owner,
	}

	resp.ResourceData = clientData
	resp.DataSourceData = clientData
}

// DataSources defines the data sources implemented in the provider.
func (p *githubxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewRepositoryDataSource,
		NewRepositoryBranchDataSource,
		NewRepositoryFileDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *githubxProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRepositoryResource,
		NewRepositoryBranchResource,
		NewRepositoryFileResource,
		NewRepositoryPullRequestAutoMergeResource,
	}
}

// getGitHubCLIToken attempts to retrieve a token from the GitHub CLI.
// Returns an empty string if gh CLI is not available or not authenticated.
func getGitHubCLIToken() string {
	// Try to get token from 'gh auth token' command
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		// gh CLI not available or not authenticated
		return ""
	}
	// Trim whitespace from the token
	token := strings.TrimSpace(string(output))
	if token != "" {
		return token
	}
	return ""
}

// getGitHubAppToken generates an installation access token for a GitHub App.
func getGitHubAppToken(ctx context.Context, appAuth *appAuthModel, baseURL *url.URL) (string, *provider.ConfigureResponse) {
	resp := &provider.ConfigureResponse{}

	appID := appAuth.ID.ValueInt64()
	installationID := appAuth.InstallationID.ValueInt64()
	pemFile := appAuth.PEMFile.ValueString()

	if appID == 0 {
		resp.Diagnostics.AddError(
			"Invalid App ID",
			"GitHub App ID must be provided and greater than 0.",
		)
		return "", resp
	}

	if installationID == 0 {
		resp.Diagnostics.AddError(
			"Invalid Installation ID",
			"GitHub App Installation ID must be provided and greater than 0.",
		)
		return "", resp
	}

	if pemFile == "" {
		resp.Diagnostics.AddError(
			"Missing PEM File",
			"GitHub App PEM file path must be provided.",
		)
		return "", resp
	}

	// Read PEM file
	pemData, err := os.ReadFile(pemFile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read PEM File",
			fmt.Sprintf("Unable to read GitHub App private key file: %v", err),
		)
		return "", resp
	}

	// Parse PEM block
	block, _ := pem.Decode(pemData)
	if block == nil {
		resp.Diagnostics.AddError(
			"Invalid PEM File",
			"Failed to decode PEM file. Ensure the file contains a valid RSA private key.",
		)
		return "", resp
	}

	// Parse RSA private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Private Key",
				fmt.Sprintf("Failed to parse private key: %v", err),
			)
			return "", resp
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			resp.Diagnostics.AddError(
				"Invalid Key Type",
				"Private key must be an RSA key.",
			)
			return "", resp
		}
	}

	// Generate JWT
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(), // Issued at time (allow 60s clock skew)
		"exp": now.Add(10 * time.Minute).Unix(),  // Expires in 10 minutes
		"iss": appID,                             // Issuer (App ID)
	})

	jwtToken, err := token.SignedString(privateKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Sign JWT",
			fmt.Sprintf("Unable to sign JWT token: %v", err),
		)
		return "", resp
	}

	// Create a temporary client with JWT to get installation token
	tempHTTPClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: jwtToken},
	))
	tempClient := github.NewClient(tempHTTPClient)

	// Set base URL if not default
	if baseURL.String() != "https://api.github.com/" {
		tempClient.BaseURL = baseURL
	}

	// Get installation token
	installationToken, _, err := tempClient.Apps.CreateInstallationToken(ctx, installationID, &github.InstallationTokenOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Installation Token",
			fmt.Sprintf("Unable to create installation access token: %v", err),
		)
		return "", resp
	}

	return installationToken.GetToken(), resp
}
