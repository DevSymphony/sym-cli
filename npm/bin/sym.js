#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

/**
 * Get the platform-specific binary name
 */
function getPlatformBinary() {
  const platform = process.platform;
  const arch = process.arch;

  const binaryMap = {
    'darwin-arm64': 'sym-darwin-arm64',
    'darwin-x64': 'sym-darwin-amd64',
    'linux-x64': 'sym-linux-amd64',
    'linux-arm64': 'sym-linux-arm64',
    'win32-x64': 'sym-windows-amd64.exe',
  };

  const key = `${platform}-${arch}`;
  const binaryName = binaryMap[key];

  if (!binaryName) {
    console.error(`Error: Unsupported platform: ${platform}-${arch}`);
    console.error('Supported platforms:');
    Object.keys(binaryMap).forEach(k => console.error(`  - ${k}`));
    process.exit(1);
  }

  return path.join(__dirname, binaryName);
}

/**
 * Main execution
 */
function main() {
  const binaryPath = getPlatformBinary();

  if (!fs.existsSync(binaryPath)) {
    console.error(`Error: Binary not found at ${binaryPath}`);
    console.error('');
    console.error('This usually means the binary download failed during installation.');
    console.error('Please try reinstalling:');
    console.error('  npm uninstall @dev-symphony/sym');
    console.error('  npm install @dev-symphony/sym');
    console.error('');
    console.error('If the problem persists, please report an issue at:');
    console.error('  https://github.com/DevSymphony/sym-cli/issues');
    process.exit(1);
  }

  // Make sure the binary is executable (Unix systems)
  if (process.platform !== 'win32') {
    try {
      fs.chmodSync(binaryPath, '755');
    } catch (err) {
      // Ignore chmod errors, might not have permission
    }
  }

  // Pass all arguments to the binary
  const args = process.argv.slice(2);

  const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    windowsHide: true,
  });

  child.on('error', (err) => {
    console.error(`Error executing binary: ${err.message}`);
    process.exit(1);
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code || 0);
    }
  });
}

main();
