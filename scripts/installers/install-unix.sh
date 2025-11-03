#!/bin/bash
# Symphony macOS/Linux Installer

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

echo -e "${BLUE}🎵 Symphony Installer${NC}"
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

BINARY_NAME="symphony-${OS}-${ARCH}"

echo -e "${GREEN}✓ Detected: $OS ($ARCH)${NC}"
echo ""

# Check if binary exists
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}❌ Error: $BINARY_NAME not found in current directory${NC}"
    echo -e "${YELLOW}   Please run this script from the dist/ directory${NC}"
    exit 1
fi

# Determine installation method
if [ "$EUID" -eq 0 ] || sudo -n true 2>/dev/null; then
    # Running as root or can sudo
    INSTALL_DIR="/usr/local/bin"
    USE_SUDO=true
    echo -e "${GREEN}✓ Will install to system directory: $INSTALL_DIR${NC}"
else
    # Install to user directory
    INSTALL_DIR="$HOME/.local/bin"
    USE_SUDO=false
    echo -e "${YELLOW}⚠ Installing to user directory: $INSTALL_DIR${NC}"
    echo -e "${YELLOW}  (Run with sudo to install system-wide)${NC}"
fi

echo ""
echo -e "${BLUE}📂 Installation directory: $INSTALL_DIR${NC}"
echo ""

# Ask for confirmation
read -p "Continue with installation? (Y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Nn]$ ]]; then
    echo -e "${RED}❌ Installation cancelled${NC}"
    exit 1
fi

# Create installation directory
echo -e "${BLUE}📁 Creating installation directory...${NC}"
if [ "$USE_SUDO" = true ]; then
    sudo mkdir -p "$INSTALL_DIR"
else
    mkdir -p "$INSTALL_DIR"
fi

# Copy binary
echo -e "${BLUE}📋 Installing Symphony binary...${NC}"
if [ "$USE_SUDO" = true ]; then
    sudo cp "$BINARY_NAME" "$INSTALL_DIR/symphony"
    sudo chmod 755 "$INSTALL_DIR/symphony"
else
    cp "$BINARY_NAME" "$INSTALL_DIR/symphony"
    chmod 755 "$INSTALL_DIR/symphony"
fi

# Configure PATH
echo ""
echo -e "${BLUE}🔧 Configuring PATH...${NC}"

PATH_CONFIGURED=false

# Check if already in PATH
if echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo -e "${GREEN}✓ $INSTALL_DIR is already in PATH${NC}"
    PATH_CONFIGURED=true
elif [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
    echo -e "${GREEN}✓ /usr/local/bin is typically in PATH by default${NC}"
    PATH_CONFIGURED=true
else
    # Need to add to PATH
    echo -e "${YELLOW}⚠ $INSTALL_DIR is not in PATH${NC}"
    echo ""
    read -p "Add to PATH automatically? (Y/n) " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        # Detect shell
        SHELL_NAME=$(basename "$SHELL")
        case "$SHELL_NAME" in
            bash)
                SHELL_RC="$HOME/.bashrc"
                ;;
            zsh)
                SHELL_RC="$HOME/.zshrc"
                ;;
            fish)
                SHELL_RC="$HOME/.config/fish/config.fish"
                ;;
            *)
                SHELL_RC="$HOME/.profile"
                ;;
        esac

        echo -e "${BLUE}📝 Adding to $SHELL_RC...${NC}"

        # Add to shell config
        if [ "$SHELL_NAME" = "fish" ]; then
            echo "" >> "$SHELL_RC"
            echo "# Symphony" >> "$SHELL_RC"
            echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$SHELL_RC"
        else
            echo "" >> "$SHELL_RC"
            echo "# Symphony" >> "$SHELL_RC"
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
        fi

        echo -e "${GREEN}✓ Added to $SHELL_RC${NC}"
        echo -e "${YELLOW}⚠ Note: Run 'source $SHELL_RC' or restart your terminal${NC}"
        PATH_CONFIGURED=true

        # Update current session
        export PATH="$INSTALL_DIR:$PATH"
    else
        echo -e "${YELLOW}⊘ Skipped PATH configuration${NC}"
        echo -e "${YELLOW}  To use Symphony, either:${NC}"
        echo -e "${YELLOW}    1. Run: $INSTALL_DIR/symphony${NC}"
        echo -e "${YELLOW}    2. Add '$INSTALL_DIR' to your PATH manually${NC}"
    fi
fi

# Verify installation
echo ""
echo -e "${BLUE}🔍 Verifying installation...${NC}"

if [ -f "$INSTALL_DIR/symphony" ]; then
    echo -e "${GREEN}✓ Binary installed successfully${NC}"

    # Try to run symphony
    if "$INSTALL_DIR/symphony" whoami --help &>/dev/null; then
        echo -e "${GREEN}✓ Symphony is executable${NC}"
    else
        echo -e "${YELLOW}⚠ Warning: Could not execute symphony${NC}"
    fi
else
    echo -e "${RED}❌ Installation failed: binary not found${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Symphony installed successfully!${NC}"
echo ""
echo -e "${BLUE}📖 Next steps:${NC}"
echo -e "  ${NC}1. Configure: ${GREEN}symphony config${NC}"
echo -e "  ${NC}2. Login:     ${GREEN}symphony login${NC}"
echo -e "  ${NC}3. Init repo: ${GREEN}symphony init${NC}"
echo -e "  ${NC}4. Dashboard: ${GREEN}symphony dashboard${NC}"
echo ""
echo -e "${BLUE}📚 Documentation: See README.md${NC}"
echo ""

if [ "$PATH_CONFIGURED" = false ]; then
    echo -e "${YELLOW}⚠ Remember to add $INSTALL_DIR to your PATH!${NC}"
    echo ""
fi
