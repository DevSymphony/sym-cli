#!/bin/bash
set -e

echo "Installing Go development tools..."
make setup

echo "Installing global npm packages..."
npm install -g @google/gemini-cli @openai/codex

echo "Setup completed successfully!"
