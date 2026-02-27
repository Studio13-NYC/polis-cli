# Contributing Windows Support to Polis CLI

## Changes Made

This branch adds Windows support to Polis CLI:

- **`docs/WINDOWS.md`**: Comprehensive Windows setup and usage guide
- **`scripts/setup-windows.ps1`**: Automated setup script for Windows
- **`README.md`**: Updated to include Windows in supported platforms

## Creating the Pull Request

### Step 1: Fork the Repository

1. Go to https://github.com/vdibart/polis-cli
2. Click the **"Fork"** button in the top right
3. This creates a fork at: `https://github.com/YOUR_USERNAME/polis-cli`

### Step 2: Add Your Fork as Remote

```bash
cd D:\Studio13\online\polis-cli

# Add your fork as a remote (replace YOUR_USERNAME)
git remote add fork https://github.com/YOUR_USERNAME/polis-cli.git

# Verify remotes
git remote -v
```

### Step 3: Push Feature Branch to Your Fork

```bash
# Push the feature branch to your fork
git push -u fork feature/windows-support
```

### Step 4: Create Pull Request

1. Go to: https://github.com/vdibart/polis-cli
2. You should see a banner suggesting to create a PR from your fork
3. Or go to **Pull requests** → **New pull request**
4. Click **"compare across forks"**
5. **Base repository**: `vdibart/polis-cli`
6. **Base branch**: `main`
7. **Head repository**: `YOUR_USERNAME/polis-cli`
8. **Compare branch**: `feature/windows-support`
9. Click **"Create pull request"**

## PR Title

```
Add Windows support via Git Bash
```

## PR Description

```markdown
## Summary

Adds Windows support to Polis CLI, enabling users to run Polis natively on Windows using Git Bash (included with Git for Windows).

## Changes

### Documentation

- **`docs/WINDOWS.md`**: Comprehensive Windows setup and usage guide
  - Git Bash installation and configuration instructions
  - Dependency installation (jq, pandoc) via multiple package managers
  - Path configuration and troubleshooting guide
  - WSL2 alternative option for full Linux compatibility

### Automation

- **`scripts/setup-windows.ps1`**: Automated Windows setup script
  - Detects and installs missing dependencies (jq, pandoc) via winget/choco/scoop
  - Automatically configures Git Bash PATH
  - Handles path detection for dependencies installed via different methods
  - Provides clear setup instructions and verification

### Updates

- **`README.md`**: Updated platform support
  - Added Windows to platform badge
  - Added Windows setup reference in prerequisites section
  - Added WINDOWS.md to documentation list

## Testing

✅ Tested on Windows 10/11 with Git Bash  
✅ Setup script verified and working  
✅ All polis commands functional (init, post, render, comment, etc.)  
✅ Dependencies (jq, pandoc) auto-detected and configured  
✅ Path handling works correctly with Windows drive letters  

## Benefits

- **Minimal setup**: Automated script handles most configuration
- **Native experience**: Works with existing Git for Windows installation
- **Full compatibility**: All polis commands work identically to Linux/macOS
- **No code changes**: Uses existing bash scripts via Git Bash
- **Alternative option**: WSL2 documented for users who prefer full Linux environment

## Usage

After setup, Windows users can use Polis CLI exactly like Linux/macOS users:

```bash
polis init --site-title "My Site"
polis post my-post.md
polis render
```

## Related

This addresses the platform limitation mentioned in the README (currently Linux | macOS only) and makes Polis accessible to Windows users without requiring WSL2 or virtual machines.
```

## Alternative: Direct Push (if you have write access)

If you have write access to the repository:

```bash
git push -u origin feature/windows-support
```

Then create PR directly on GitHub.
