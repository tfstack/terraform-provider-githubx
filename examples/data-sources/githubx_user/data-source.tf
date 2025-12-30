terraform {
  required_providers {
    githubx = {
      source  = "registry.terraform.io/tfstack/githubx"
      version = "0.1.0"
    }
  }
}

provider "githubx" {
  # Token can be provided here or via GITHUB_TOKEN environment variable
  # token = "your-github-token-here"
}

data "githubx_user" "octocat" {
  username = "octocat"
}

output "user_id" {
  value = data.githubx_user.octocat.user_id
}

output "user_name" {
  value = data.githubx_user.octocat.name
}

output "user_login" {
  value = data.githubx_user.octocat.username
}

output "user_bio" {
  value = data.githubx_user.octocat.bio
}

output "user_public_repos" {
  value = data.githubx_user.octocat.public_repos
}
