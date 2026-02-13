#!/bin/bash
#
# Cleanup Script: Remove secrets.yaml from Git History
#
# ⚠️  WARNING: This script REWRITES Git history
# - ALL team members must re-clone the repository after this
# - Coordinate with team BEFORE running
# - Backup your work before execution
#
# Prerequisites:
# - git-filter-repo installed (recommended) OR git filter-branch
# - Team notification and coordination完成

set -e  # Exit on error

echo "================================================"
echo "Git History Cleanup: Remove secrets.yaml"
echo "================================================"
echo ""
echo "⚠️  WARNING: This will rewrite Git history!"
echo "   All team members must re-clone after this operation."
echo ""
read -p "Have you coordinated with the team? (yes/no): " confirmed

if [ "$confirmed" != "yes" ]; then
    echo "Aborted. Please coordinate with team first."
    exit 1
fi

echo ""
echo "Creating backup branch..."
BACKUP_BRANCH="backup-before-cleanup-$(date +%Y%m%d-%H%M%S)"
git branch "$BACKUP_BRANCH"
echo "✅ Backup created: $BACKUP_BRANCH"

echo ""
echo "Checking for git-filter-repo..."
if command -v git-filter-repo &> /dev/null; then
    echo "✅ git-filter-repo found (recommended method)"

    # Method 1: git-filter-repo (fast and safe)
    echo ""
    echo "Removing secrets.yaml from entire history..."
    git filter-repo --path deploy/k8s/secrets.yaml --invert-paths --force

else
    echo "⚠️  git-filter-repo not found, using git filter-branch (slower)"
    echo "   Install git-filter-repo: pip install git-filter-repo"
    echo ""

    # Method 2: git filter-branch (slower, but built-in)
    echo "Removing secrets.yaml from entire history..."
    git filter-branch --force --index-filter \
        'git rm --cached --ignore-unmatch deploy/k8s/secrets.yaml' \
        --prune-empty --tag-name-filter cat -- --all

    # Clean up filter-branch artifacts
    echo ""
    echo "Cleaning up filter-branch artifacts..."
    rm -rf .git/refs/original/
    git reflog expire --expire=now --all
    git gc --prune=now --aggressive
fi

echo ""
echo "Re-adding secrets.yaml template..."
# Re-add the current template version
git checkout HEAD -- deploy/k8s/secrets.yaml 2>/dev/null || true
if [ ! -f deploy/k8s/secrets.yaml ]; then
    echo "⚠️  secrets.yaml not found, you may need to restore it manually"
else
    git add deploy/k8s/secrets.yaml
    git commit -m "chore: re-add secrets.yaml as template (no real values)

This file was removed from Git history to prevent credential leakage.
Real values should be stored in secrets.prod.yaml (gitignored).

See: doc/code-review/2026-02-13-adversarial-code-review.md (P0-1)"
fi

echo ""
echo "================================================"
echo "✅ Git history cleanup completed!"
echo "================================================"
echo ""
echo "Next steps:"
echo ""
echo "1. Verify the cleanup:"
echo "   git log --all -- deploy/k8s/secrets.yaml"
echo "   (Should only show the re-add commit)"
echo ""
echo "2. Force push to remote (DESTRUCTIVE!):"
echo "   git push origin --force --all"
echo "   git push origin --force --tags"
echo ""
echo "3. Notify all team members:"
echo "   - Delete their local clones"
echo "   - Re-clone the repository"
echo "   - Do NOT try to merge or pull"
echo ""
echo "4. Verify remote:"
echo "   git clone <repo-url> /tmp/verify-clone"
echo "   cd /tmp/verify-clone"
echo "   git log --all -- deploy/k8s/secrets.yaml"
echo ""
echo "Backup branch: $BACKUP_BRANCH"
echo "To restore: git reset --hard $BACKUP_BRANCH"
echo ""
