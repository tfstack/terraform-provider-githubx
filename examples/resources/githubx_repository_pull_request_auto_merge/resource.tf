terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Configure the provider
provider "githubx" {
  owner = "cloudbuildlab" # Optional: set your GitHub username or organization. If not set, will use authenticated user.
  # Token can be provided here or via GITHUB_TOKEN environment variable
  # token = "your-github-token-here"
}

# First, create a repository
resource "githubx_repository" "example" {
  name        = "my-pr-example-repo"
  description = "Repository for pull request examples"
  visibility  = "public"
  auto_init   = true # Initialize with README to create default branch
}

# Create a feature branch for the PR
resource "githubx_repository_branch" "feature" {
  repository    = githubx_repository.example.name
  branch        = "feature/new-feature"
  source_branch = "main"
}

# Add some files to the feature branch
resource "githubx_repository_file" "feature_file1" {
  repository = githubx_repository.example.name
  branch     = githubx_repository_branch.feature.branch
  file       = "feature/new-file.md"
  content    = "# New Feature\n\nThis is a new feature file added via Terraform."
}

resource "githubx_repository_file" "feature_file2" {
  repository = githubx_repository.example.name
  branch     = githubx_repository_branch.feature.branch
  file       = "feature/config.json"
  content = jsonencode({
    feature = "enabled"
    version = "1.0.0"
  })
}

# Example 1: Basic pull request (no auto-merge)
# resource "githubx_repository_pull_request_auto_merge" "basic_pr" {
#   repository = githubx_repository.example.name
#   base_ref   = "main"
#   head_ref   = githubx_repository_branch.feature.branch
#   title      = "Add new feature"
#   body       = "This PR adds a new feature with multiple files."
#   # Note: Without merge_when_ready = true, this PR will be created but not merged
# }

# Example 2: Pull request with auto-merge (waits for checks and approvals)
# Note: Only one PR can exist per base_ref/head_ref pair, so comment out Example 1 to use this
resource "githubx_repository_pull_request_auto_merge" "auto_merge_pr" {
  repository         = githubx_repository.example.name
  base_ref           = "main"
  head_ref           = githubx_repository_branch.feature.branch
  title              = "Auto-merge feature PR"
  body               = "This PR will be automatically merged when ready."
  merge_when_ready   = true
  merge_method       = "merge" # Use "merge" as default (or "squash"/"rebase" if allowed by repository settings)
  wait_for_checks    = true
  auto_delete_branch = true
}
