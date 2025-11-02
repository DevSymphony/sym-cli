#!/bin/bash

# 간단한 액세스 체크 스크립트

# 현재 사용자 정보
USER=$(symphony whoami --json 2>/dev/null | jq -r '.username')
ROLE=$(symphony my-role --json 2>/dev/null | jq -r '.role')
REPO=$(symphony my-role --json 2>/dev/null | jq -r '"\(.owner)/\(.repo)"')

if [ -z "$USER" ] || [ "$USER" == "null" ]; then
    echo "❌ Not logged in"
    echo "Run: symphony login"
    exit 1
fi

echo "Access Check Report"
echo "==================="
echo "User:       $USER"
echo "Repository: $REPO"
echo "Role:       $ROLE"
echo ""

# 역할별 권한 표시
case "$ROLE" in
    "admin")
        echo "Permissions:"
        echo "  ✓ Read files"
        echo "  ✓ Write files"
        echo "  ✓ Deploy to production"
        echo "  ✓ Manage roles"
        ;;
    "developer")
        echo "Permissions:"
        echo "  ✓ Read files"
        echo "  ✓ Write files"
        echo "  ✓ Deploy to staging"
        echo "  ✗ Deploy to production"
        echo "  ✗ Manage roles"
        ;;
    "viewer")
        echo "Permissions:"
        echo "  ✓ Read files"
        echo "  ✗ Write files"
        echo "  ✗ Deploy"
        echo "  ✗ Manage roles"
        ;;
    "none")
        echo "⚠️  No permissions assigned"
        echo "Contact an admin to get access"
        ;;
esac
