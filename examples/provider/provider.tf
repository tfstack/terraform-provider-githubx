terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

provider "githubx" {}

data "githubx_user" "example" {
  username = "octocat"
}

output "user" {
  value = data.githubx_user.example
}
