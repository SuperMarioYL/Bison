#!/bin/bash
# Bison v0.0.1 Release Script
# Run this script to create a complete release

set -e

echo "🚀 Bison v0.0.1 Release"
echo "======================"
echo ""

# Step 1: Verify we're on main branch and up to date
echo "📋 Step 1: Verifying branch status..."
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "❌ Error: Not on main branch (currently on: $CURRENT_BRANCH)"
    echo "   Please switch to main branch first:"
    echo "   git checkout main"
    exit 1
fi

echo "✅ On main branch"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "⚠️  Warning: You have uncommitted changes"
    echo "   Please commit or stash them first"
    git status --short
    exit 1
fi

echo "✅ No uncommitted changes"
echo ""

# Step 2: Create Git tag
echo "📋 Step 2: Creating Git tag v0.0.1..."
git tag -a v0.0.1 -m "Bison v0.0.1 - Initial Release

🎉 First official release of Bison GPU Resource Billing Platform

✨ Features:
- Multi-tenant GPU management with Capsule integration
- Real-time billing powered by OpenCost
- Modern React dashboard with Apple-style design
- Zero external database (ConfigMaps only)
- Multi-platform Docker images (amd64/arm64)
- Complete REST API for automation

📦 Components:
- API Server v0.0.1 (Go 1.24)
- Web UI v0.0.1 (React 18)
- Helm Chart v0.0.1
- Documentation Site (Docusaurus 3.9.2)

📚 Documentation: https://supermarioyl.github.io/Bison/
🐛 Issues: https://github.com/SuperMarioYL/Bison/issues
"

echo "✅ Tag created: v0.0.1"
echo ""

# Step 3: Push tag to remote
echo "📋 Step 3: Pushing tag to GitHub..."
echo "   This will trigger GitHub Actions to:"
echo "   - Build multi-platform Docker images"
echo "   - Package Helm chart"
echo "   - Create GitHub Release"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cancelled. To push manually later:"
    echo "   git push origin v0.0.1"
    exit 1
fi

git push origin v0.0.1

echo "✅ Tag pushed to remote"
echo ""

# Step 4: Package Helm chart locally (for manual verification)
echo "📋 Step 4: Packaging Helm chart..."
helm package ./deploy/charts/bison

if [ -f "bison-0.0.1.tgz" ]; then
    echo "✅ Helm chart packaged: bison-0.0.1.tgz"
    echo "   Size: $(du -h bison-0.0.1.tgz | cut -f1)"
else
    echo "❌ Failed to package Helm chart"
    exit 1
fi
echo ""

# Step 5: Create GitHub Release (if gh CLI is available)
if command -v gh &> /dev/null; then
    echo "📋 Step 5: Creating GitHub Release..."
    echo ""
    read -p "Create GitHub Release now? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        gh release create v0.0.1 \
            --title "Bison v0.0.1 - Initial Release 🎉" \
            --notes-file RELEASE_v0.0.1.md \
            --latest \
            bison-0.0.1.tgz

        echo "✅ GitHub Release created!"
        echo "   View at: https://github.com/SuperMarioYL/Bison/releases/tag/v0.0.1"
    else
        echo "⏭️  Skipped. Create manually at:"
        echo "   https://github.com/SuperMarioYL/Bison/releases/new?tag=v0.0.1"
        echo "   Attach file: bison-0.0.1.tgz"
        echo "   Use content from: RELEASE_v0.0.1.md"
    fi
else
    echo "📋 Step 5: GitHub Release (manual)"
    echo "   gh CLI not found. Create release manually:"
    echo ""
    echo "   1. Visit: https://github.com/SuperMarioYL/Bison/releases/new?tag=v0.0.1"
    echo "   2. Title: Bison v0.0.1 - Initial Release 🎉"
    echo "   3. Copy content from: RELEASE_v0.0.1.md"
    echo "   4. Attach: bison-0.0.1.tgz"
    echo "   5. Check: 'Set as the latest release'"
    echo "   6. Click: 'Publish release'"
fi
echo ""

# Step 6: Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Release Process Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📦 What happens next:"
echo ""
echo "   1. GitHub Actions will automatically:"
echo "      • Build Docker images for linux/amd64 & linux/arm64"
echo "      • Push to ghcr.io/supermarioyl/bison/*"
echo "      • Package and publish Helm chart"
echo "      • Deploy documentation to GitHub Pages"
echo ""
echo "   2. Verify release artifacts (in ~10 minutes):"
echo "      • Docker images:"
echo "        docker pull ghcr.io/supermarioyl/bison/api-server:0.0.1"
echo "        docker pull ghcr.io/supermarioyl/bison/web-ui:0.0.1"
echo ""
echo "      • Helm chart:"
echo "        helm repo add bison https://supermarioyl.github.io/Bison/charts/"
echo "        helm repo update"
echo "        helm search repo bison"
echo ""
echo "      • Documentation:"
echo "        open https://supermarioyl.github.io/Bison/"
echo ""
echo "   3. Announce release:"
echo "      • GitHub Discussions: https://github.com/SuperMarioYL/Bison/discussions"
echo "      • Update README badges with v0.0.1"
echo "      • Share on social media / community channels"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🎉 Thank you for using Bison!"
echo ""
