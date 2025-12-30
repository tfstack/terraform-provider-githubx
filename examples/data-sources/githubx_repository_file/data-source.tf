terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Example 1: Using full_name to read a file
data "githubx_repository_file" "example_full_name" {
  full_name = "cloudbuildlab/.github"
  file      = "README.md"
  branch    = "main"
}

output "file_content" {
  value = data.githubx_repository_file.example_full_name.content
}

output "file_sha" {
  value = data.githubx_repository_file.example_full_name.sha
}

output "file_ref" {
  value = data.githubx_repository_file.example_full_name.ref
}

output "file_commit_sha" {
  value = data.githubx_repository_file.example_full_name.commit_sha
}

output "file_commit_message" {
  value = data.githubx_repository_file.example_full_name.commit_message
}

output "file_commit_author" {
  value = data.githubx_repository_file.example_full_name.commit_author
}

# Example 2: Using repository with provider-level owner
provider "githubx" {
  owner = "cloudbuildlab"
}

data "githubx_repository_file" "example_repository" {
  repository = "actions-markdown-lint"
  file       = ".github/workflows/lint.yml"
  branch     = "main"
}

output "file_id" {
  value = data.githubx_repository_file.example_repository.id
}

output "file_content_from_repo" {
  value = data.githubx_repository_file.example_repository.content
}

output "file_sha_from_repo" {
  value = data.githubx_repository_file.example_repository.sha
}

output "file_commit_email" {
  value = data.githubx_repository_file.example_repository.commit_email
}

# Example 3: Reading file from default branch (branch not specified)
data "githubx_repository_file" "example_default_branch" {
  full_name = "cloudbuildlab/.github"
  file      = "CONTRIBUTING.md"
}

output "file_from_default_branch" {
  value = data.githubx_repository_file.example_default_branch.content
}
