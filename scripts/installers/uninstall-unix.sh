#!/bin/bash
# Symphony macOS/Linux Uninstaller

set -e

# Set UTF-8 encoding for proper emoji and Korean display
export LC_ALL="${LC_ALL:-en_US.UTF-8}"
export LANG="${LANG:-en_US.UTF-8}"
export LANGUAGE="${LANGUAGE:-en_US:en}"

# Ensure terminal supports UTF-8
if [ -z "$LC_ALL" ] && [ -z "$LANG" ]; then
    export LC_ALL=C.UTF-8
    export LANG=C.UTF-8
fi

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🎵 Symphony Uninstaller${NC}"
echo ""

# Check possible installation locations
POSSIBLE_LOCATIONS=(
    "/usr/local/bin/symphony"
    "$HOME/.local/bin/symphony"
)

FOUND_LOCATIONS=()
for loc in "${POSSIBLE_LOCATIONS[@]}"; do
    if [ -f "$loc" ]; then
        FOUND_LOCATIONS+=("$loc")
    fi
done

if [ ${#FOUND_LOCATIONS[@]} -eq 0 ]; then
    echo -e "${RED}❌ Symphony installation not found${NC}"
    echo ""
    echo -e "${YELLOW}Checked locations:${NC}"
    for loc in "${POSSIBLE_LOCATIONS[@]}"; do
        echo -e "${YELLOW}  - $loc${NC}"
    done
    exit 1
fi

echo -e "${BLUE}📂 Found Symphony installations:${NC}"
for loc in "${FOUND_LOCATIONS[@]}"; do
    echo -e "  ${NC}- $loc"
done
echo ""

# Ask for confirmation
read -p "Uninstall Symphony? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}❌ Uninstallation cancelled${NC}"
    exit 1
fi

# Remove installations
for binary_path in "${FOUND_LOCATIONS[@]}"; do
    echo ""
    echo -e "${BLUE}🗑️  Removing $binary_path...${NC}"

    if [ -w "$binary_path" ]; then
        rm -f "$binary_path"
        echo -e "${GREEN}✓ Removed${NC}"
    else
        # Try with sudo
        if sudo rm -f "$binary_path" 2>/dev/null; then
            echo -e "${GREEN}✓ Removed (with sudo)${NC}"
        else
            echo -e "${RED}❌ Failed to remove${NC}"
            echo -e "${YELLOW}  Try: sudo rm $binary_path${NC}"
        fi
    fi
done

# Clean PATH entries from shell configs
echo ""
echo -e "${BLUE}🔧 Cleaning shell configuration files...${NC}"

SHELL_CONFIGS=(
    "$HOME/.bashrc"
    "$HOME/.zshrc"
    "$HOME/.profile"
    "$HOME/.config/fish/config.fish"
)

for config in "${SHELL_CONFIGS[@]}"; do
    if [ -f "$config" ]; then
        # Check if file contains Symphony PATH entries
        if grep -q "# Symphony" "$config" 2>/dev/null; then
            # Create backup
            cp "$config" "$config.backup"

            # Remove Symphony lines
            sed -i.tmp '/# Symphony/,+1d' "$config" 2>/dev/null || \
            sed -i '' '/# Symphony/,+1d' "$config" 2>/dev/null

            echo -e "${GREEN}✓ Cleaned $(basename $config)${NC}"
        fi
    fi
done

# Optional: Remove config files
echo ""
read -p "Remove configuration files in ~/.config/symphony? (y/N) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    CONFIG_DIR="$HOME/.config/symphony"
    if [ -d "$CONFIG_DIR" ]; then
        rm -rf "$CONFIG_DIR"
        echo -e "${GREEN}✓ Configuration files removed${NC}"
    else
        echo -e "${YELLOW}⊘ No configuration files found${NC}"
    fi
fi

echo ""
echo -e "${GREEN}✅ Symphony uninstalled successfully!${NC}"
echo ""
echo -e "${YELLOW}⚠ You may need to restart your terminal or run:${NC}"
echo -e "${YELLOW}  source ~/.bashrc  (or ~/.zshrc, etc.)${NC}"
echo ""
