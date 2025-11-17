# Setup GitHub Authentication

Configure GitHub CLI authentication with Personal Access Token (PAT) for automated workflows.

## Usage

```
/setup-github [--check-only]
```

## Options

- `--check-only` - Check current authentication status without making changes

## What This Command Does

Guides you through setting up GitHub authentication using a Personal Access Token (PAT), which is required for:

- ✅ Creating repositories and GitHub Projects
- ✅ Managing issues and pull requests via API
- ✅ Configuring repository settings programmatically
- ✅ Running automated workflows (CI/CD, priority reviews)
- ✅ Using slash commands that interact with GitHub

## Why PAT Instead of OAuth?

**Personal Access Tokens (PAT)** are better for automation because:

1. **No Browser Required** - Works in CI/CD and automated scripts
2. **Fine-Grained Permissions** - Control exactly what access is granted
3. **Token Rotation** - Easy to revoke and regenerate
4. **Multiple Tokens** - Different tokens for different purposes
5. **Secure Storage** - Can be stored in environment variables or secrets

## Process

### Step 1: Check Current Authentication

```bash
gh auth status
```

**If Already Authenticated**:
```
✓ Logged in to github.com as username
✓ Git operations for github.com configured to use https protocol
✓ Token: gho_****
```

**If Not Authenticated**:
```
✗ Not logged in to github.com
```

### Step 2: Choose Authentication Method

The command will present options:

```
GitHub Authentication Setup
═══════════════════════════════════════════

Current Status: Not authenticated

Choose authentication method:
  1. Create new PAT (recommended for automation)
  2. Use existing PAT
  3. OAuth web flow (for manual use only)
  4. Exit without changes

Enter choice [1-4]:
```

### Step 3: Create Personal Access Token

If you choose option 1 or 2, you'll need a GitHub PAT.

#### Creating a New PAT (Classic)

1. **Navigate to GitHub**:
   ```
   https://github.com/settings/tokens
   ```

2. **Click "Generate new token"** → **"Generate new token (classic)"**

3. **Configure Token**:
   - **Note**: `git-template automation (created YYYY-MM-DD)`
   - **Expiration**: 90 days (recommended) or No expiration (less secure)
   - **Select scopes**:
     - ✅ `repo` (Full control of private repositories)
     - ✅ `workflow` (Update GitHub Actions workflows)
     - ✅ `admin:org` → `read:org` (Read org and team membership)
     - ✅ `project` (Full control of projects)
     - ✅ `delete_repo` (Delete repositories - optional)

4. **Generate Token** and **copy it immediately** (you won't see it again!)

#### Creating a Fine-Grained PAT (Recommended)

1. **Navigate to GitHub**:
   ```
   https://github.com/settings/tokens?type=beta
   ```

2. **Click "Generate new token"**

3. **Configure Token**:
   - **Token name**: `git-template automation`
   - **Expiration**: 90 days (recommended)
   - **Repository access**:
     - "All repositories" (for template automation)
     - OR "Only select repositories" (for specific projects)

   - **Permissions** → **Repository permissions**:
     - ✅ Administration: Read and write
     - ✅ Contents: Read and write
     - ✅ Issues: Read and write
     - ✅ Metadata: Read-only (automatic)
     - ✅ Projects: Read and write
     - ✅ Pull requests: Read and write
     - ✅ Workflows: Read and write

   - **Permissions** → **Organization permissions** (if using org):
     - ✅ Projects: Read and write
     - ✅ Members: Read-only

4. **Generate token** and **copy it**

### Step 4: Authenticate with Token

**Option 1: Interactive Entry**

```bash
gh auth login --with-token
# Paste your token and press Enter
# Press Ctrl+D to finish
```

**Option 2: Environment Variable**

```bash
export GITHUB_TOKEN="ghp_your_token_here"
gh auth login --with-token <<< "$GITHUB_TOKEN"
```

**Option 3: From File**

```bash
echo "ghp_your_token_here" > ~/.github_token
chmod 600 ~/.github_token
gh auth login --with-token < ~/.github_token
```

### Step 5: Verify Authentication

```bash
gh auth status
```

**Expected Output**:
```
✓ Logged in to github.com as username
✓ Git operations for github.com configured to use https protocol
✓ Token: ghp_****
✓ Token scopes: repo, workflow, read:org, project
```

### Step 6: Configure Git Credentials

```bash
gh auth setup-git
```

This configures Git to use the GitHub CLI as a credential helper.

## Security Best Practices

### Token Storage

**NEVER commit tokens to version control!**

Add to `.gitignore`:
```
.env
.github_token
*.token
credentials.json
```

**Secure Storage Options**:

1. **Environment Variables** (Recommended for CI/CD):
   ```bash
   # Add to ~/.bashrc or ~/.zshrc
   export GITHUB_TOKEN="ghp_your_token_here"
   ```

2. **GitHub CLI Keyring** (Automatic):
   ```bash
   gh auth login --with-token
   # Token stored securely in system keyring
   ```

3. **Secrets Manager** (Production):
   - AWS Secrets Manager
   - Azure Key Vault
   - HashiCorp Vault
   - 1Password CLI

### Token Rotation

**Rotate tokens regularly** (every 90 days recommended):

```bash
# 1. Create new token with same permissions
# 2. Update environment variable or keyring
export GITHUB_TOKEN="ghp_new_token_here"
gh auth login --with-token <<< "$GITHUB_TOKEN"

# 3. Verify new token works
gh auth status

# 4. Revoke old token at:
# https://github.com/settings/tokens
```

### Token Revocation

**If token is compromised**:

1. **Revoke immediately** at https://github.com/settings/tokens
2. **Generate new token** with different permissions
3. **Update all services** using the old token
4. **Audit recent activity** for unauthorized access

## Troubleshooting

### Token Authentication Failed

```
Error: authentication failed
```

**Solutions**:
1. Verify token hasn't expired
2. Check token has required scopes
3. Ensure token is for correct GitHub account
4. Try pasting token again (no extra whitespace)

### Insufficient Permissions

```
Error: Resource not accessible by integration
```

**Solutions**:
1. Verify token has required scopes (see Step 3)
2. Check repository/org permissions
3. Generate new token with additional scopes
4. Contact repository/org admin for access

### Token Not Found

```
Error: GITHUB_TOKEN not set
```

**Solutions**:
1. Set environment variable: `export GITHUB_TOKEN="..."`
2. Or use `gh auth login --with-token`
3. Add to shell profile for persistence

### Multiple GitHub Accounts

If you have multiple GitHub accounts:

```bash
# Check current account
gh auth status

# Switch accounts
gh auth logout
gh auth login --with-token
# Enter token for different account

# Or use hostname for GitHub Enterprise
gh auth login --hostname github.company.com
```

## Automated Setup Script

For automated environments (CI/CD):

```bash
#!/bin/bash
# setup-github-auth.sh

set -e

if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GITHUB_TOKEN environment variable not set"
  exit 1
fi

# Authenticate with token
echo "$GITHUB_TOKEN" | gh auth login --with-token

# Verify authentication
gh auth status

# Configure git
gh auth setup-git

echo "✓ GitHub authentication configured"
```

**Usage in CI/CD**:

```yaml
# .github/workflows/ci.yml
- name: Setup GitHub CLI
  run: |
    echo "${{ secrets.GITHUB_TOKEN }}" | gh auth login --with-token
    gh auth setup-git
```

## Integration with Commands

Once authenticated, these commands will work:

### Repository Management
- `/create-repo` - Creates repository and GitHub Project
- `/configure-repo` - Configures repository settings

### Issue Management
- `/spec-to-issue` - Creates issues and adds to GitHub Project
- `/create-spec-issue` - Creates tracking issues

### Project Management
- `/prioritize` - Updates issue priorities in GitHub Project
- `/review-priorities` - Automated priority reviews

## Token Scopes Reference

### Required Scopes (Minimum)

- ✅ `repo` - Repository access
- ✅ `project` - GitHub Projects access
- ✅ `workflow` - GitHub Actions workflows

### Optional Scopes (Recommended)

- ✅ `read:org` - Read organization membership
- ✅ `delete_repo` - Delete repositories (for cleanup)
- ✅ `admin:repo_hook` - Manage webhooks

### Fine-Grained Permissions Mapping

| Classic Scope | Fine-Grained Permission |
|---------------|------------------------|
| `repo` | Contents: Read and write |
| `repo` | Issues: Read and write |
| `repo` | Pull requests: Read and write |
| `project` | Projects: Read and write |
| `workflow` | Workflows: Read and write |
| `read:org` | Members: Read-only |
| `admin:org` → `write:org` (projects) | Organization projects: Admin |

## Verification Checklist

After setup, verify:

- [ ] `gh auth status` shows authenticated
- [ ] `gh repo list` works (shows your repositories)
- [ ] `gh project list` works (shows your projects)
- [ ] `gh issue list` works (shows issues in test repo)
- [ ] Token has expiration date set (security)
- [ ] Token is NOT committed to version control
- [ ] Token is stored securely (keyring or env var)

## Example: Complete Setup Flow

```bash
# Step 1: Check current status
$ /setup-github --check-only
Status: Not authenticated

# Step 2: Create PAT at https://github.com/settings/tokens
# Copy token: ghp_abc123...

# Step 3: Authenticate
$ export GITHUB_TOKEN="ghp_abc123..."
$ gh auth login --with-token <<< "$GITHUB_TOKEN"
✓ Authenticated as username

# Step 4: Configure git
$ gh auth setup-git
✓ Configured git credential helper

# Step 5: Verify
$ gh auth status
✓ Logged in to github.com as username
✓ Token: ghp_****
✓ Token scopes: repo, workflow, read:org, project

# Step 6: Test
$ gh repo list
username/repo1
username/repo2

# Ready to use automated commands!
$ /create-repo test-project "Testing automation"
```

## Terminal Demo

Watch the demo: `asciinema play demos/commands/setup-github.cast`

## Related Commands

- `/create-repo` - Creates repository (requires authentication)
- `/configure-repo` - Configures repository settings
- `/spec-to-issue` - Creates issues in GitHub Project
- `/prioritize` - Updates GitHub Project priorities
- `/review-priorities` - Automated priority management

## Security Notice

⚠️ **CRITICAL**: Never share or commit your GitHub Personal Access Token!

- Tokens grant access to your GitHub account
- Treat tokens like passwords
- Rotate tokens regularly (every 90 days)
- Revoke immediately if compromised
- Use separate tokens for different purposes

## Resources

- **GitHub PAT Documentation**: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
- **Fine-Grained PAT**: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token
- **GitHub CLI Documentation**: https://cli.github.com/manual/
- **Token Scopes Reference**: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps

---

**Security First**: Always prioritize token security. When in doubt, rotate your token.
