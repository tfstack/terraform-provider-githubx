# Terraform Provider for GitHubx

A supplemental Terraform provider offering extended GitHub capabilities alongside the official provider, built using the Terraform Plugin Framework.

## Features

- **User Information**: Query GitHub user information and profiles
- **Repository Management**: Create and manage GitHub repositories with extended capabilities
- **Branch Management**: Create and manage repository branches
- **File Management**: Create, update, and manage files in repositories
- **Pull Request Automation**: Create pull requests with auto-merge capabilities, including automatic approval and merge when ready
- **Data Sources**: Query GitHub users, repositories, branches, and files
- **Extensible**: Designed to complement the official GitHub provider with additional capabilities

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider plugin)

## Installation

### Using Terraform Registry

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

provider "githubx" {
  token = var.github_token
}
```

### Building from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/tfstack/terraform-provider-githubx.git
   cd terraform-provider-githubx
   ```

2. Build the provider:

   ```bash
   go install
   ```

3. Install the provider to your local Terraform plugins directory:

   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tfstack/githubx/0.1.0/linux_amd64
   cp $GOPATH/bin/terraform-provider-githubx ~/.terraform.d/plugins/registry.terraform.io/tfstack/githubx/0.1.0/linux_amd64/
   ```

## Configuration

The provider supports various configuration options for authentication, GitHub Enterprise Server, and development/testing.

## Authentication

The provider supports multiple authentication methods for higher rate limits and access to private resources.

### Option 1: Provider Configuration Block (Personal Access Token)

```hcl
provider "githubx" {
  token = "your-github-token-here"
}
```

### Option 2: OAuth Token

```hcl
provider "githubx" {
  oauth_token = "your-oauth-token-here"
}
```

### Option 3: Environment Variable

Set the `GITHUB_TOKEN` environment variable:

```bash
export GITHUB_TOKEN="your-github-token-here"
```

Then use the provider without the token attribute:

```hcl
provider "githubx" {
  # token will be read from GITHUB_TOKEN environment variable
}
```

### Option 4: GitHub CLI Authentication

If you have the GitHub CLI (`gh`) installed and authenticated, the provider will automatically use your GitHub CLI token:

```bash
# Authenticate with GitHub CLI (if not already done)
gh auth login
```

Then use the provider without any configuration:

```hcl
provider "githubx" {
  # token will be automatically retrieved from 'gh auth token'
}
```

### Option 5: GitHub App Authentication

For GitHub App authentication, you need to provide the App ID, Installation ID, and path to the private key PEM file:

```hcl
provider "githubx" {
  app_auth {
    id             = 123456
    installation_id = 789012
    pem_file       = "/path/to/private-key.pem"
  }
}
```

**Note:** The provider checks for authentication in this order:

1. Provider `token` attribute (Personal Access Token)
2. Provider `oauth_token` attribute
3. `GITHUB_TOKEN` environment variable
4. GitHub CLI (`gh auth token`)
5. GitHub App authentication (`app_auth` block)
6. Unauthenticated (if none of the above are available)

**Development Container:** If you're using the devcontainer, the host OS GitHub CLI authentication is automatically mounted and will be used by the provider. You only need to authenticate once on your host OS with `gh auth login`, and it will persist across container rebuilds.

### Getting a Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token" (classic) or "Generate new token (fine-grained)"
3. Select the scopes/permissions you need
4. Click "Generate token"
5. Copy the token immediately (it won't be shown again)

**Note**: For most data sources, a token is optional but recommended. Without a token, you'll be limited to 60 requests/hour. With a token, you get 5,000 requests/hour.

### GitHub App Authentication

GitHub App authentication is useful for CI/CD pipelines and automated workflows. To use it:

1. **Create a GitHub App** in your organization or personal account
2. **Install the App** in the repositories or organization where you need access
3. **Generate a private key** for the App (download the PEM file)
4. **Configure the provider** with the App ID, Installation ID, and path to the PEM file

The provider will automatically:

- Generate a JWT token using the private key
- Exchange it for an installation access token
- Use the installation token for API requests

**Note**: Installation tokens are automatically refreshed as needed (they expire after 1 hour).

## Provider Configuration Options

### Base URL (GitHub Enterprise Server)

The provider supports GitHub Enterprise Server (GHES) by configuring a custom base URL:

```hcl
provider "githubx" {
  base_url = "https://github.example.com/api/v3/"
}
```

Or via environment variable:

```bash
export GITHUB_BASE_URL="https://github.example.com/api/v3/"
```

**Default:** `https://api.github.com/`

### Owner Configuration

Specify the GitHub owner (user or organization) to manage:

```hcl
provider "githubx" {
  owner = "my-organization"
}
```

Or via environment variable:

```bash
export GITHUB_OWNER="my-organization"
```

This can be useful for:

- Multi-organization scenarios
- Resource scoping
- Validation and defaults

### Insecure Mode (TLS)

Enable insecure mode for testing with self-signed certificates:

```hcl
provider "githubx" {
  insecure = true
}
```

Or via environment variable:

```bash
export GITHUB_INSECURE="true"
```

**Warning:** Only use this in development/testing environments. It disables TLS certificate verification.

### Rate Limits

- **Unauthenticated**: 60 requests/hour
- **Authenticated**: 5,000 requests/hour

The provider will work without a token, but with very limited rate limits. For testing, this is usually sufficient for a few queries.

## Quick Start

Here's a simple example to get you started:

```hcl
terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

provider "githubx" {
  # Token will be read from GITHUB_TOKEN environment variable
}

data "githubx_user" "example" {
  username = "octocat"
}

output "user" {
  value = data.githubx_user.example
}
```

For more examples, see the [`examples/`](examples/) directory.

## Data Sources

- [`githubx_user`](docs/data-sources/user.md) - Retrieves information about a GitHub user
- [`githubx_repository`](docs/data-sources/repository.md) - Retrieves information about a GitHub repository
- [`githubx_repository_branch`](docs/data-sources/repository_branch.md) - Retrieves information about a GitHub repository branch
- [`githubx_repository_file`](docs/data-sources/repository_file.md) - Retrieves information about a file in a GitHub repository

## Resources

- [`githubx_repository`](docs/resources/repository.md) - Creates and manages a GitHub repository
- [`githubx_repository_branch`](docs/resources/repository_branch.md) - Creates and manages a GitHub repository branch
- [`githubx_repository_file`](docs/resources/repository_file.md) - Creates and manages files in a GitHub repository
- [`githubx_repository_pull_request_auto_merge`](docs/resources/repository_pull_request_auto_merge.md) - Creates and manages a GitHub pull request with optional auto-merge capabilities

## Local Testing (Development Container)

When developing in the devcontainer, you can test the provider locally using the following steps:

### 1. Build the Provider

Build the provider binary:

```bash
make build
# or
go build -o terraform-provider-githubx -buildvcs=false
```

### 2. Install Provider Locally

Install the provider to Terraform's local plugin directory so Terraform can find it:

Option A: Using Make (Recommended)

```bash
make install-local
```

Option B: Manual installation

```bash
# Create the plugin directory structure
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tfstack/githubx/0.1.0/linux_amd64

# Copy the built binary
cp terraform-provider-githubx ~/.terraform.d/plugins/registry.terraform.io/tfstack/githubx/0.1.0/linux_amd64/
```

**Note:** The version number (`0.1.0`) should match the version in your Terraform configuration's `required_providers` block.

### 3. Initialize Examples (Automated)

Option A: Initialize all examples automatically

```bash
make init-examples
```

This will:

- Build and install the provider locally
- Initialize Terraform in all example directories
- Skip examples that require variables (you'll need to set those manually)

Option B: Initialize a specific example

```bash
make init-example EXAMPLE=examples/data-sources/githubx_user
```

Option C: Manual initialization

Navigate to the example directory and initialize manually:

```bash
cd examples/data-sources/githubx_user
terraform init
```

### 4. Test with Example Configuration

After initialization, navigate to any example directory and test the provider:

```bash
cd examples/data-sources/githubx_user

# Option 1: Use .env file (recommended - edit .env with your values)
# Copy .env.example to .env and fill in your values, then:
source .env

# Option 2: Set environment variables manually
# Set your GitHub token (optional but recommended)
export GITHUB_TOKEN="your-github-token-here"

# Plan to see what Terraform will do
terraform plan

# Apply to test the provider
terraform apply
```

You should see output like:

```text
Outputs:

user_bio = "GitHub's mascot"
user_id = 583231
user_login = "octocat"
user_name = "The Octocat"
user_public_repos = 8
```

### 5. Run Unit Tests

Run the unit tests:

```bash
make test
# or
go test -v ./...
```

### 6. Run Test Coverage

Generate a test coverage report:

Option A: Using Make (Recommended)

```bash
make test-coverage
```

This will:

- Run tests with coverage
- Display coverage summary in the terminal
- Generate an HTML coverage report (`coverage.html`)

Option B: Manual commands

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage report in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage for specific package
go test -cover ./internal/provider/
```

**Coverage Options:**

- `-coverprofile=coverage.out` - Generate coverage profile file
- `-covermode=count` - Show how many times each statement was executed (default: `set`)
- `-covermode=atomic` - Same as count but thread-safe (useful for parallel tests)
- `-coverpkg=./...` - Include coverage for all packages, not just tested ones

**Example output:**

```text
github.com/tfstack/terraform-provider-githubx/internal/provider/data_source_user.go:Metadata    100.0%
github.com/tfstack/terraform-provider-githubx/internal/provider/data_source_user.go:Schema      100.0%
...
total:                                                                    (statements)    85.5%
```

### 7. Run Acceptance Tests

Acceptance tests make real API calls to GitHub. Set the `TF_ACC` environment variable to enable them:

```bash
export GITHUB_TOKEN="your-github-token-here"
export TF_ACC=1
make testacc
# or
TF_ACC=1 go test -v ./...
```

**Warning:** Acceptance tests make real API calls to GitHub. Use a test token and be mindful of rate limits.

### 8. Quick Setup Scripts

Helper scripts are available to automate common tasks:

**Install Provider Locally:**

```bash
make install-local
```

**Initialize All Examples:**

```bash
make init-examples
```

**Initialize Specific Example:**

```bash
make init-example EXAMPLE=examples/data-sources/githubx_user
```

### Troubleshooting

- **Provider not found:** Ensure the version in your Terraform config matches the directory version (`0.1.0`)
- **Permission denied:** Make sure the plugin directory is writable: `chmod -R 755 ~/.terraform.d/plugins/`
- **Provider version mismatch:** Update the version in your Terraform config or rename the plugin directory to match
- **Rate limit errors:** Set the `GITHUB_TOKEN` environment variable for higher rate limits (5,000 requests/hour vs 60 requests/hour)
- **Connection errors:** Verify your token is correct and has the necessary permissions

## Examples

Comprehensive examples are available in the [`examples/`](examples/) directory:

- **Data Sources**: See [`examples/data-sources/`](examples/data-sources/) for examples of querying GitHub resources
  - `githubx_user` - Query user information
  - `githubx_repository` - Query repository information
  - `githubx_repository_branch` - Query branch information
  - `githubx_repository_file` - Query file content and metadata
- **Resources**: See [`examples/resources/`](examples/resources/) for examples of managing GitHub resources
  - `githubx_repository` - Create and manage repositories
  - `githubx_repository_branch` - Create and manage branches
  - `githubx_repository_file` - Create and manage files
  - `githubx_repository_pull_request_auto_merge` - Create pull requests with auto-merge
- **Provider**: See [`examples/provider/`](examples/provider/) for a simple provider example

Each example includes a `data-source.tf`, `resource.tf`, or `provider.tf` file with working Terraform configuration.

## Limitations

- **Rate Limits**: Without authentication, you're limited to 60 requests/hour. Use a token for 5,000 requests/hour.
- **Scope**: This provider is designed to supplement the official GitHub provider, not replace it. Use it for extended capabilities alongside the official provider.

## Documentation

Full documentation for all data sources and resources is available in the [`docs/`](docs/) directory:

- [Data Sources Documentation](docs/data-sources/)
- [Resources Documentation](docs/resources/)

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for information on developing the provider.

## Support

- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/tfstack/terraform-provider-githubx/issues)
- **Discussions**: Ask questions and share ideas on [GitHub Discussions](https://github.com/tfstack/terraform-provider-githubx/discussions)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
