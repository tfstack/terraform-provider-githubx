terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Example 1: Using full_name
data "githubx_repository_branch" "example_full_name" {
  full_name = "cloudbuildlab/.github"
  branch    = "main"
}

output "branch_ref" {
  value = data.githubx_repository_branch.example_full_name.ref
}

output "branch_sha" {
  value = data.githubx_repository_branch.example_full_name.sha
}

output "branch_etag" {
  value = data.githubx_repository_branch.example_full_name.etag
}

# Example 2: Using repository with provider-level owner
provider "githubx" {
  owner = "cloudbuildlab"
}

data "githubx_repository_branch" "example_repository" {
  repository = "actions-markdown-lint"
  branch     = "main"
}

output "branch_id" {
  value = data.githubx_repository_branch.example_repository.id
}

output "branch_ref_from_repo" {
  value = data.githubx_repository_branch.example_repository.ref
}

output "branch_sha_from_repo" {
  value = data.githubx_repository_branch.example_repository.sha
}
