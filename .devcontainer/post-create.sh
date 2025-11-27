#!/bin/bash
set -e

echo "Installing Go development tools..."
make setup

echo "Installing global npm packages..."
npm install -g @google/gemini-cli @openai/codex

echo "Installing Python venv package for Pylint adapter..."
sudo apt-get update -qq && sudo apt-get install -y python3-venv

echo "Setup completed successfully!"
