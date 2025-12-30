terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Configure the provider
# The owner can be set via provider config, environment variable GITHUB_OWNER, or will default to authenticated user
provider "githubx" {
  # owner = "cloudbuildlab" # Optional: set your GitHub username or organization. If not set, will use authenticated user.
  # Token can be provided here or via GITHUB_TOKEN environment variable
  # token = "your-github-token-here"
}

# First, create a repository
resource "githubx_repository" "example" {
  name        = "my-branch-example-repo"
  description = "Repository for branch examples"
  visibility  = "public"
  auto_init   = true # Initialize with README to create default branch
}

# Example 1: Create a branch from default source (main)
resource "githubx_repository_branch" "develop" {
  repository = githubx_repository.example.name
  branch     = "develop"
}

output "develop_branch_ref" {
  value = githubx_repository_branch.develop.ref
}

output "develop_branch_sha" {
  value = githubx_repository_branch.develop.sha
}

# Example 2: Create a branch from a specific source branch
# Note: This depends on the develop branch being created first
resource "githubx_repository_branch" "feature" {
  repository    = githubx_repository.example.name
  branch        = "feature/new-feature"
  source_branch = "develop"
  depends_on    = [githubx_repository_branch.develop]
}

output "feature_branch_ref" {
  value = githubx_repository_branch.feature.ref
}

# Example 3: Create a branch from a specific commit SHA
# Note: Replace the SHA with an actual commit SHA from your repository
resource "githubx_repository_branch" "hotfix" {
  repository = githubx_repository.example.name
  branch     = "hotfix/critical-fix"
  # source_sha = "abc123def456..." # Uncomment and replace with actual commit SHA
  depends_on = [githubx_repository.example]
}

output "hotfix_branch_sha" {
  value = githubx_repository_branch.hotfix.sha
}
