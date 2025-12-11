# Publishing heapcheck to GitHub

## Quick Start (5 minutes)

### Step 1: Create GitHub Repository

1. Go to https://github.com/new
2. Repository name: `heapcheck`
3. Description: `Human-friendly Go escape analysis CLI with actionable suggestions`
4. Public repository
5. **DO NOT** initialize with README (we have one)
6. Click "Create repository"

### Step 2: Update Username in Files

Replace `harshakonda` with your actual GitHub username in these files:

```bash
# On macOS/Linux, run this (replace YOUR_GITHUB_USERNAME):
export GITHUB_USER="YOUR_GITHUB_USERNAME"

# Update all files
sed -i.bak "s/harshakonda/$GITHUB_USER/g" README.md
sed -i.bak "s/harshakonda/$GITHUB_USER/g" CONTRIBUTING.md
sed -i.bak "s/harshakonda/$GITHUB_USER/g" .goreleaser.yaml
sed -i.bak "s/github.com/anthropic/heapcheck/github.com/$GITHUB_USER/heapcheck/g" go.mod
sed -i.bak "s/github.com/anthropic/heapcheck/github.com/$GITHUB_USER/heapcheck/g" cmd/heapcheck/main.go
sed -i.bak "s/github.com/anthropic/heapcheck/github.com/$GITHUB_USER/heapcheck/g" internal/reporter/reporter.go
sed -i.bak "s/github.com/anthropic/heapcheck/github.com/$GITHUB_USER/heapcheck/g" internal/categorizer/categorizer.go

# Remove backup files
rm -f *.bak **/*.bak

# Verify changes
grep -r "harshakonda" . --include="*.go" --include="*.md" --include="*.yaml"
grep -r "anthropic" . --include="*.go" --include="*.md" --include="*.yaml"
```

### Step 3: Initialize Git and Push

```bash
cd heapcheck

# Initialize git
git init
git add .
git commit -m "Initial commit: heapcheck v0.1.0

Human-friendly Go escape analysis CLI with:
- 19 escape categories with actionable suggestions
- Text, JSON, HTML, and SARIF output formats
- CI/CD integration support
"

# Add remote and push
git branch -M main
git remote add origin https://github.com/harshakonda/heapcheck.git
git push -u origin main
```

### Step 4: Create First Release

```bash
# Tag the release
git tag -a v0.1.0 -m "Initial release

Features:
- Parse Go compiler escape analysis output
- Categorize escapes into 19 categories
- Provide actionable optimization suggestions
- Support text, JSON, HTML, and SARIF output
- CI/CD integration with GitHub Actions
"

# Push the tag (triggers GitHub Actions release)
git push origin v0.1.0
```

### Step 5: Verify

1. Check GitHub Actions: `https://github.com/harshakonda/heapcheck/actions`
2. Check Releases: `https://github.com/harshakonda/heapcheck/releases`
3. Test installation:
   ```bash
   go install github.com/harshakonda/heapcheck/cmd/heapcheck@latest
   heapcheck --help
   ```

---

## Post-Publishing Checklist

- [ ] GitHub repo created and code pushed
- [ ] v0.1.0 release created with binaries
- [ ] README badges working (CI, Go Report Card)
- [ ] `go install` works
- [ ] SARIF upload to GitHub Code Scanning works

## Optional: Homebrew Tap

1. Create repo `homebrew-tap` on GitHub
2. GoReleaser will auto-publish formula on release
3. Users can then: `brew install harshakonda/tap/heapcheck`

## Promotion Ideas

1. **Reddit**: Post to r/golang
2. **Twitter/X**: Tag @golang
3. **Hacker News**: "Show HN: heapcheck - Human-friendly Go escape analysis"
4. **Dev.to**: Write a blog post
5. **Go Weekly**: Submit to newsletter

---

## Troubleshooting

### GoReleaser fails
- Check `.goreleaser.yaml` syntax
- Ensure `GITHUB_TOKEN` has write permissions

### Tests fail in CI
- Run `go test ./...` locally first
- Check Go version compatibility

### SARIF upload fails
- Ensure Code Scanning is enabled in repo settings
- Check SARIF file format with: `./heapcheck --format=sarif ./... | head -50`
