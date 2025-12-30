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
  name        = "my-file-example-repo"
  description = "Repository for file examples"
  visibility  = "public"
  auto_init   = true # Initialize with README to create default branch
}

# Example 1: Create a simple file on the default branch
# Note: The repository is auto-initialized with a README.md, so we'll create a different file
resource "githubx_repository_file" "docs" {
  repository = githubx_repository.example.name
  file       = "docs/GETTING_STARTED.md"
  content    = "# Getting Started\n\nThis is a getting started guide managed by Terraform."
}

output "docs_commit_sha" {
  value = githubx_repository_file.docs.commit_sha
}

output "docs_sha" {
  value = githubx_repository_file.docs.sha
}

# Example 2: Create a file on a specific branch
resource "githubx_repository_branch" "develop" {
  repository = githubx_repository.example.name
  branch     = "develop"
  depends_on = [githubx_repository.example]
}

resource "githubx_repository_file" "config" {
  repository = githubx_repository.example.name
  file       = ".github/config.yml"
  branch     = githubx_repository_branch.develop.branch
  content    = "repository:\n  name: ${githubx_repository.example.name}\n  description: ${githubx_repository.example.description}\n"
  depends_on = [githubx_repository_branch.develop]
}

output "config_file_ref" {
  value = githubx_repository_file.config.ref
}

# Example 3: Create a file with custom commit message and author
resource "githubx_repository_file" "license" {
  repository     = githubx_repository.example.name
  file           = "LICENSE"
  content        = "MIT License\n\nCopyright (c) 2024\n"
  commit_message = "Add MIT license file"
  commit_author  = "Terraform Bot"
  commit_email   = "terraform@example.com"
}

output "license_commit_message" {
  value = githubx_repository_file.license.commit_message
}

# Example 4: Create a file with auto-create branch feature
resource "githubx_repository_file" "feature_file" {
  repository                      = githubx_repository.example.name
  file                            = "feature/new-feature.md"
  branch                          = "feature/new-feature"
  content                         = "# New Feature\n\nThis is a new feature file."
  autocreate_branch               = true
  autocreate_branch_source_branch = "main"
  depends_on                      = [githubx_repository.example]
}

output "feature_file_id" {
  value = githubx_repository_file.feature_file.id
}

# Example 5: Overwrite existing file (created outside Terraform or by auto-init)
# Note: The repository auto_init creates a README.md, so we can overwrite it
# This demonstrates overwrite_on_create for files that exist but aren't managed by Terraform
resource "githubx_repository_file" "readme" {
  repository          = githubx_repository.example.name
  file                = "README.md"
  content             = "# My Example Repository\n\nThis is an updated README managed by Terraform.\n\n## Features\n\n- Feature 1\n- Feature 2\n"
  overwrite_on_create = true
  depends_on          = [githubx_repository.example]
}

output "readme_sha" {
  value = githubx_repository_file.readme.sha
}

# Example 6: Update an existing file (modify the same resource)
# This shows how to update a file by changing the content in the same resource
resource "githubx_repository_file" "changelog" {
  repository = githubx_repository.example.name
  file       = "CHANGELOG.md"
  content    = "# Changelog\n\n## [Unreleased]\n\n### Added\n- Initial version\n"
}

output "changelog_commit_sha" {
  value = githubx_repository_file.changelog.commit_sha
}
