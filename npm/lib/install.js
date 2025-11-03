#!/usr/bin/env node

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const { HttpsProxyAgent } = require('https-proxy-agent');

const GITHUB_REPO = 'DevSymphony/sym-cli';
const VERSION = require('../package.json').version;

/**
 * Get platform-specific asset name
 */
function getAssetName() {
  const platform = process.platform;
  const arch = process.arch;

  const assetMap = {
    'darwin-arm64': 'sym-darwin-arm64',
    'darwin-x64': 'sym-darwin-amd64',
    'linux-x64': 'sym-linux-amd64',
    'linux-arm64': 'sym-linux-arm64',
    'win32-x64': 'sym-windows-amd64.exe',
  };

  const key = `${platform}-${arch}`;
  const assetName = assetMap[key];

  if (!assetName) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return assetName;
}

/**
 * Download file from URL with redirect support
 */
function downloadFile(url, destPath, followRedirect = true) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const isHttps = urlObj.protocol === 'https:';
    const httpModule = isHttps ? https : http;

    const options = {
      method: 'GET',
      headers: {
        'User-Agent': 'sym-cli-installer',
      },
    };

    // Support proxy
    const proxy = process.env.HTTPS_PROXY || process.env.https_proxy ||
                  process.env.HTTP_PROXY || process.env.http_proxy;
    if (proxy && isHttps) {
      options.agent = new HttpsProxyAgent(proxy);
    }

    const request = httpModule.get(url, options, (response) => {
      // Handle redirects
      if (response.statusCode === 301 || response.statusCode === 302 || response.statusCode === 307 || response.statusCode === 308) {
        if (!followRedirect) {
          reject(new Error('Too many redirects'));
          return;
        }
        const redirectUrl = response.headers.location;
        if (!redirectUrl) {
          reject(new Error('Redirect without location'));
          return;
        }
        resolve(downloadFile(redirectUrl, destPath, false));
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
        return;
      }

      const file = fs.createWriteStream(destPath);
      const totalBytes = parseInt(response.headers['content-length'] || '0', 10);
      let downloadedBytes = 0;

      response.on('data', (chunk) => {
        downloadedBytes += chunk.length;
        if (totalBytes > 0) {
          const percent = ((downloadedBytes / totalBytes) * 100).toFixed(1);
          process.stdout.write(`\rDownloading: ${percent}%`);
        }
      });

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        if (totalBytes > 0) {
          process.stdout.write('\n');
        }
        console.log('Download complete');
        resolve();
      });

      file.on('error', (err) => {
        fs.unlink(destPath, () => {});
        reject(err);
      });
    });

    request.on('error', (err) => {
      reject(err);
    });

    request.setTimeout(60000, () => {
      request.destroy();
      reject(new Error('Download timeout'));
    });
  });
}

/**
 * Main installation function
 */
async function install() {
  try {
    // Skip installation in development mode
    if (process.env.NODE_ENV === 'development') {
      console.log('Skipping binary download in development mode');
      return;
    }

    const assetName = getAssetName();
    const url = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${assetName}`;
    const binDir = path.join(__dirname, '..', 'bin');
    const binaryPath = path.join(binDir, assetName);

    // Create bin directory if it doesn't exist
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Check if binary already exists
    if (fs.existsSync(binaryPath)) {
      console.log('Binary already exists, skipping download');
      return;
    }

    console.log(`Downloading Symphony CLI v${VERSION} for ${process.platform}-${process.arch}...`);
    console.log(`URL: ${url}`);

    await downloadFile(url, binaryPath);

    // Make executable on Unix systems
    if (process.platform !== 'win32') {
      fs.chmodSync(binaryPath, '755');
    }

    console.log(`Successfully installed Symphony CLI to ${binaryPath}`);
  } catch (error) {
    console.error('Installation failed:', error.message);
    console.error('');
    console.error('Please try the following:');
    console.error('1. Check your internet connection');
    console.error('2. Verify that the release v' + VERSION + ' exists at:');
    console.error(`   https://github.com/${GITHUB_REPO}/releases/tag/v${VERSION}`);
    console.error('3. If you are behind a proxy, set the HTTPS_PROXY environment variable');
    console.error('4. Report the issue at: https://github.com/' + GITHUB_REPO + '/issues');
    console.error('');
    process.exit(1);
  }
}

// Run installation
install();
