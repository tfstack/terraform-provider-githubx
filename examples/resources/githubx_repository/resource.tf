terraform {
  required_providers {
    githubx = {
      source  = "tfstack/githubx"
      version = "~> 0.1"
    }
  }
}

# Configure the provider with owner
# The owner can be set via environment variable GITHUB_OWNER instead
provider "githubx" {
  owner = "cloudbuildlab" # Replace with your GitHub username or organization
}

# Example 1: Basic public repository
resource "githubx_repository" "basic" {
  name        = "my-basic-repo"
  description = "A basic public repository"
  visibility  = "public"
}

output "basic_repository_url" {
  value = githubx_repository.basic.html_url
}

# Example 2: Private repository with features enabled
resource "githubx_repository" "private" {
  name         = "my-private-repo"
  description  = "A private repository with features enabled"
  visibility   = "private"
  has_issues   = true
  has_projects = true
  has_wiki     = true
}

output "private_repository_url" {
  value = githubx_repository.private.html_url
}

# Example 3: Repository with merge settings
resource "githubx_repository" "merge_settings" {
  name                        = "my-merge-repo"
  description                 = "Repository with custom merge settings"
  visibility                  = "public"
  allow_merge_commit          = true
  allow_squash_merge          = true
  allow_rebase_merge          = false
  allow_auto_merge            = true
  delete_branch_on_merge      = true
  squash_merge_commit_title   = "PR_TITLE"
  squash_merge_commit_message = "PR_BODY" # Valid combination with PR_TITLE
  merge_commit_title          = "PR_TITLE"
  merge_commit_message        = "PR_BODY"
}

output "merge_settings_repository_url" {
  value = githubx_repository.merge_settings.html_url
}

# Example 4: Repository with topics
resource "githubx_repository" "with_topics" {
  name        = "my-topics-repo"
  description = "Repository with topics"
  visibility  = "public"
  topics      = ["terraform", "github", "automation", "infrastructure"]
}

output "topics_repository_url" {
  value = githubx_repository.with_topics.html_url
}

# Example 5: Repository with GitHub Pages
resource "githubx_repository" "with_pages" {
  name        = "my-pages-repo"
  description = "Repository with GitHub Pages enabled"
  visibility  = "public"

  pages = {
    source = {
      branch = "main"
      path   = "/"
    }
    build_type = "legacy"
  }
}

output "pages_repository_url" {
  value = githubx_repository.with_pages.html_url
}

# Example 6: Template repository
resource "githubx_repository" "template" {
  name        = "my-template-repo"
  description = "A template repository"
  visibility  = "public"
  is_template = true
}

output "template_repository_url" {
  value = githubx_repository.template.html_url
}

# Example 7: Repository with vulnerability alerts
resource "githubx_repository" "secure" {
  name                 = "my-secure-repo"
  description          = "Repository with security features"
  visibility           = "public"
  vulnerability_alerts = true
}

output "secure_repository_url" {
  value = githubx_repository.secure.html_url
}

# # Example 8: Repository that archives on destroy
# # NOTE: When archive_on_destroy = true, the repository is archived (not deleted) when destroyed.
# # If you try to apply this again after destroying, it will fail because the repository already exists.
# # You would need to manually unarchive the repository in GitHub or use a different name.
# resource "githubx_repository" "archivable" {
#   name               = "my-archivable-repo"
#   description        = "Repository that archives instead of deleting"
#   visibility         = "private"
#   archive_on_destroy = true
# }

# output "archivable_repository_url" {
#   value = githubx_repository.archivable.html_url
# }

# Example 9: Complete repository with all features
resource "githubx_repository" "complete" {
  name                        = "my-complete-repo"
  description                 = "A complete repository example with all features"
  homepage_url                = "https://example.com"
  visibility                  = "public"
  has_issues                  = true
  has_discussions             = true
  has_projects                = true
  has_downloads               = true
  has_wiki                    = true
  allow_merge_commit          = true
  allow_squash_merge          = true
  allow_rebase_merge          = true
  allow_auto_merge            = true
  allow_update_branch         = true
  delete_branch_on_merge      = true
  squash_merge_commit_title   = "PR_TITLE"
  squash_merge_commit_message = "PR_BODY"
  merge_commit_title          = "PR_TITLE"
  merge_commit_message        = "PR_BODY"
  topics                      = ["terraform", "github", "example"]
  vulnerability_alerts        = true

  pages = {
    source = {
      branch = "main"
      path   = "/docs"
    }
    build_type = "workflow"
  }
}

output "complete_repository_url" {
  value = githubx_repository.complete.html_url
}

output "complete_repository_full_name" {
  value = githubx_repository.complete.full_name
}

output "complete_repository_default_branch" {
  value = githubx_repository.complete.default_branch
}
