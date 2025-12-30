#!/bin/bash
# Don't use set -e to allow script to continue even if some steps fail
# set -e

export DEBIAN_FRONTEND=noninteractive
export GIT_TERMINAL_PROMPT=0

# Ensure we're running as root (no sudo needed)
if [ "$(id -u)" -ne 0 ]; then
    echo "⚠️  Warning: Not running as root. Some operations may fail."
fi

echo "🚀 Setting up Terraform Provider GitHubx development environment..."

# Install bash-completion if not already installed
if ! command -v bash-completion &> /dev/null && [ ! -f /usr/share/bash-completion/bash_completion ]; then
    echo "📦 Installing bash-completion..."
    apt-get update -y && apt-get install -y bash-completion && rm -rf /var/lib/apt/lists/*
fi

# Setup bashrc with aliases and history settings
echo "⚙️  Configuring bash environment..."
cat >> /root/.bashrc << 'BASHRC_EOF'

# Devcontainer bashrc configuration
# History and completion settings

# Enable arrow key history navigation
set -o emacs
bind "\e[A": history-search-backward
bind "\e[B": history-search-forward

# History settings
HISTCONTROL=ignoredups:erasedups
HISTSIZE=10000
HISTFILESIZE=20000
shopt -s histappend

# Save and reload history after each command
PROMPT_COMMAND="history -a; history -c; history -r; $PROMPT_COMMAND"

# Enable bash completion
if [ -f /usr/share/bash-completion/bash_completion ]; then
    . /usr/share/bash-completion/bash_completion
elif [ -f /etc/bash_completion ]; then
    . /etc/bash_completion
fi

# Additional useful aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias ...='cd ../..'

# Git aliases
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline --graph --decorate'

# Terraform aliases
alias tf='terraform'
alias tfi='terraform init'
alias tfa='terraform apply'
alias tfaa='terraform apply -auto-approve'
alias tfp='terraform plan'
alias tfd='terraform destroy'
alias tfda='terraform destroy -auto-approve'

alias rmtl='rm -rf .terraform.lock.hcl'

BASHRC_EOF
echo "✅ Bash environment configured"

# Display system information
echo "📋 System Information:"
uname -a
if command -v go &> /dev/null; then
    echo "Go version: $(go version)"
    echo "Go path: $(go env GOPATH)"
else
    echo "⚠️  Go not found - will be installed by devcontainer feature"
fi

# Verify Terraform installation
echo "🔧 Verifying Terraform..."
if command -v terraform &> /dev/null; then
    echo "✅ Terraform installed: $(terraform version)"
else
    echo "⚠️  Terraform not found - will be installed by devcontainer feature"
fi

# Setup GitHub CLI authentication from host OS
echo "🔧 Setting up GitHub CLI authentication..."
if command -v gh &> /dev/null; then
    echo "✅ GitHub CLI installed: $(gh --version | head -n 1)"

    # Ensure the config directory exists
    GH_CONFIG_DIR="/root/.config/gh"
    mkdir -p "${GH_CONFIG_DIR}"

    # Check if host config is mounted (from devcontainer mount)
    # The devcontainer.json should mount ${localEnv:HOME}/.config/gh to /root/.config/gh
    if [ -d "${GH_CONFIG_DIR}" ] && [ -n "$(ls -A ${GH_CONFIG_DIR} 2>/dev/null)" ]; then
        echo "📁 Found mounted GitHub CLI config from host OS"
        # Ensure proper permissions (config files should be readable)
        chmod -R u+rw "${GH_CONFIG_DIR}" 2>/dev/null || true
        find "${GH_CONFIG_DIR}" -type f -name "*.yaml" -exec chmod 600 {} \; 2>/dev/null || true

        # Verify authentication
        if gh auth status &> /dev/null; then
            echo "✅ GitHub CLI is authenticated (using host OS auth)"
            gh auth status 2>&1 | head -n 3 || true
        else
            echo "⚠️  GitHub CLI config found but not authenticated."
            echo "   Please run 'gh auth login' on your host OS to authenticate."
        fi
    else
        echo "⚠️  GitHub CLI config not found at ${GH_CONFIG_DIR}"
        echo "   The devcontainer should mount your host's ~/.config/gh directory."
        echo "   If the mount failed, ensure you have authenticated with 'gh auth login' on your host OS."
        echo "   You can also run 'gh auth login' inside the container, but it won't persist across rebuilds."

        # Check if auth works anyway (might be using a different method)
        if gh auth status &> /dev/null; then
            echo "✅ GitHub CLI is authenticated (using alternative method)"
            gh auth status 2>&1 | head -n 3 || true
        fi
    fi
else
    echo "⚠️  GitHub CLI not found"
fi

# Install Terraform Plugin Framework docs generator
echo "📚 Installing Terraform Plugin Framework documentation generator..."
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest || echo "⚠️  Failed to install tfplugindocs (may retry later)"

# Verify Go tools (non-blocking, may not be installed yet)
echo "🔧 Verifying Go tools..."
command -v golangci-lint >/dev/null && echo "golangci-lint: $(golangci-lint version)" || echo "⚠️  golangci-lint not found"
command -v goimports >/dev/null && echo "goimports: $(which goimports)" || echo "⚠️  goimports not found"
command -v gopls >/dev/null && echo "gopls: $(which gopls)" || echo "⚠️  gopls not found"

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
cd /workspaces/terraform-provider-githubx || cd /workspace
go mod download || echo "⚠️  Go mod download failed (may retry later)"
go mod verify || echo "⚠️  Go mod verify failed"

# Build the provider to verify everything works
echo "🔨 Building provider..."
go build -buildvcs=false -o terraform-provider-githubx || echo "⚠️  Build failed (may retry later)"

# Install provider locally for Terraform to use (only if build succeeded)
echo "📦 Installing provider locally for Terraform..."
if [ -f terraform-provider-githubx ]; then
VERSION="0.1.0"
PLATFORM="linux_amd64"
PLUGIN_DIR="${HOME}/.terraform.d/plugins/registry.terraform.io/tfstack/githubx/${VERSION}/${PLATFORM}"
mkdir -p "${PLUGIN_DIR}"
    cp terraform-provider-githubx "${PLUGIN_DIR}/" && echo "✅ Provider installed to ${PLUGIN_DIR}" || echo "⚠️  Failed to install provider"
else
    echo "⚠️  Provider binary not found, skipping installation"
fi

# Initialize Terraform in examples (non-blocking, may fail if variables needed)
echo "🔧 Initializing Terraform examples..."
for dir in examples/data-sources/*/ examples/resources/*/ examples/provider/; do
	if [ -f "${dir}data-source.tf" ] || [ -f "${dir}resource.tf" ] || [ -f "${dir}provider.tf" ] || [ -f "${dir}main.tf" ] || [ -f "${dir}"*.tf ]; then
		echo "  Initializing ${dir}..."
		(cd "${dir}" && terraform init -upgrade -input=false > /dev/null 2>&1 && echo "    ✅ ${dir} initialized" || echo "    ⚠️  ${dir} skipped (may need variables)")
	fi
	done

# Load .env file if it exists
echo "🔐 Loading environment variables from .env file..."
if [ -f /workspaces/terraform-provider-githubx/.env ]; then
    set -a
    source /workspaces/terraform-provider-githubx/.env
    set +a
    echo "✅ Environment variables loaded from .env"
elif [ -f /workspace/.env ]; then
    set -a
    source /workspace/.env
    set +a
    echo "✅ Environment variables loaded from .env"
else
    echo "⚠️  No .env file found. Create one from .env.example if needed."
fi

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Available commands:"
echo "  make build          - Build the provider"
echo "  make install        - Install the provider"
echo "  make install-local   - Install provider locally for Terraform testing"
echo "  make init-examples  - Initialize Terraform in all examples"
echo "  make init-example   - Initialize a specific example (EXAMPLE=path)"
echo "  make test           - Run tests"
echo "  make fmt            - Format code"
echo "  make docs           - Generate documentation"
echo ""
echo "💡 The provider is already installed locally and examples are initialized!"
echo "   Navigate to any example directory and run 'terraform plan' or 'terraform apply'."
