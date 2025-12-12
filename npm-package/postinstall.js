#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execFileSync } = require('child_process');

const VERSION = process.env.CONTEXTD_VERSION || 'latest';
const BIN_DIR = path.join(__dirname, 'bin');
const BINARY_NAME = process.platform === 'win32' ? 'contextd.exe' : 'contextd';
const BINARY_PATH = path.join(BIN_DIR, BINARY_NAME);

// Platform mapping
const PLATFORM_MAP = {
  'darwin-arm64': 'darwin_arm64',
  'darwin-x64': 'darwin_amd64',
  'linux-x64': 'linux_amd64',
  'linux-arm64': 'linux_arm64',
};

function getPlatformKey() {
  const platform = process.platform;
  const arch = process.arch;
  return `${platform}-${arch}`;
}

async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: '/repos/fyrsmithlabs/contextd/releases/latest',
      headers: { 'User-Agent': 'contextd-mcp-npm' }
    };

    https.get(options, (response) => {
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to get latest release: HTTP ${response.statusCode}`));
        return;
      }

      let data = '';
      response.on('data', (chunk) => data += chunk);
      response.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release.tag_name);
        } catch (e) {
          reject(new Error('Failed to parse release data'));
        }
      });
      response.on('error', reject);
    }).on('error', reject);
  });
}

async function getDownloadUrl(version) {
  const platformKey = getPlatformKey();
  const platformSuffix = PLATFORM_MAP[platformKey];

  if (!platformSuffix) {
    throw new Error(`Unsupported platform: ${platformKey}. Supported: ${Object.keys(PLATFORM_MAP).join(', ')}`);
  }

  const baseUrl = 'https://github.com/fyrsmithlabs/contextd/releases';

  // Get actual version for latest
  let actualVersion = version;
  if (version === 'latest') {
    actualVersion = await getLatestVersion();
    console.log(`Latest version: ${actualVersion}`);
  }

  // Remove 'v' prefix if present for the filename
  const versionForFile = actualVersion.replace(/^v/, '');

  return `${baseUrl}/download/${actualVersion}/contextd_${versionForFile}_${platformSuffix}.tar.gz`;
}

function downloadFile(url) {
  return new Promise((resolve, reject) => {
    const follow = (url, redirects = 0) => {
      if (redirects > 5) {
        reject(new Error('Too many redirects'));
        return;
      }

      https.get(url, (response) => {
        if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          follow(response.headers.location, redirects + 1);
          return;
        }

        if (response.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${response.statusCode}`));
          return;
        }

        const chunks = [];
        response.on('data', (chunk) => chunks.push(chunk));
        response.on('end', () => resolve(Buffer.concat(chunks)));
        response.on('error', reject);
      }).on('error', reject);
    };

    follow(url);
  });
}

function extractTarGz(tempFile, destDir) {
  // Use execFileSync with explicit arguments to avoid shell injection
  execFileSync('tar', ['-xzf', tempFile, '-C', destDir], { stdio: 'inherit' });
}

async function main() {
  // Skip if binary already exists and CONTEXTD_FORCE_DOWNLOAD is not set
  if (fs.existsSync(BINARY_PATH) && !process.env.CONTEXTD_FORCE_DOWNLOAD) {
    console.log('contextd binary already exists, skipping download');
    return;
  }

  const platformKey = getPlatformKey();
  if (!PLATFORM_MAP[platformKey]) {
    console.error(`\n⚠️  Unsupported platform: ${platformKey}`);
    console.error(`   Supported platforms: ${Object.keys(PLATFORM_MAP).join(', ')}`);
    console.error(`   Please install contextd manually: brew install fyrsmithlabs/tap/contextd\n`);
    process.exit(0); // Don't fail install, just warn
  }

  const url = await getDownloadUrl(VERSION);
  console.log(`Downloading contextd for ${platformKey}...`);
  console.log(`URL: ${url}`);

  // Create bin directory
  if (!fs.existsSync(BIN_DIR)) {
    fs.mkdirSync(BIN_DIR, { recursive: true });
  }

  const tempFile = path.join(BIN_DIR, 'temp.tar.gz');

  try {
    const buffer = await downloadFile(url);
    console.log(`Downloaded ${(buffer.length / 1024 / 1024).toFixed(2)} MB`);

    // Write to temp file
    fs.writeFileSync(tempFile, buffer);

    // Extract
    extractTarGz(tempFile, BIN_DIR);

    // Make binary executable
    if (process.platform !== 'win32') {
      fs.chmodSync(BINARY_PATH, 0o755);
    }

    console.log(`✓ contextd installed to ${BINARY_PATH}`);
  } catch (error) {
    console.error(`\n⚠️  Failed to download contextd: ${error.message}`);
    console.error(`   Please install manually: brew install fyrsmithlabs/tap/contextd\n`);
    process.exit(0); // Don't fail install, just warn
  } finally {
    // Clean up temp file
    if (fs.existsSync(tempFile)) {
      fs.unlinkSync(tempFile);
    }
  }
}

main();
