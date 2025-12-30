terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Example 1: Using full_name
data "githubx_repository" "example_full_name" {
  full_name = "cloudbuildlab/.github"
}

output "repository_full_name" {
  value = data.githubx_repository.example_full_name.full_name
}

output "repository_description" {
  value = data.githubx_repository.example_full_name.description
}

output "repository_default_branch" {
  value = data.githubx_repository.example_full_name.default_branch
}

output "repository_html_url" {
  value = data.githubx_repository.example_full_name.html_url
}

# Example 2: Using name with provider-level owner
provider "githubx" {
  owner = "cloudbuildlab"
}

data "githubx_repository" "example_name" {
  name = "actions-markdown-lint"
}

output "repository_name" {
  value = data.githubx_repository.example_name.name
}

output "repository_visibility" {
  value = data.githubx_repository.example_name.visibility
}

output "repository_primary_language" {
  value = data.githubx_repository.example_name.primary_language
}

output "repository_topics" {
  value = data.githubx_repository.example_name.topics
}
