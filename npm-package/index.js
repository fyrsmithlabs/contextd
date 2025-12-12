#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const BIN_DIR = path.join(__dirname, 'bin');
const BINARY_NAME = process.platform === 'win32' ? 'contextd.exe' : 'contextd';
const BINARY_PATH = path.join(BIN_DIR, BINARY_NAME);

// Check if binary exists
if (!fs.existsSync(BINARY_PATH)) {
  console.error('contextd binary not found. Running postinstall to download...');
  require('./postinstall.js');

  // Check again after postinstall
  if (!fs.existsSync(BINARY_PATH)) {
    console.error('Failed to install contextd binary.');
    console.error('Please install manually: brew install fyrsmithlabs/tap/contextd');
    process.exit(1);
  }
}

// Pass all arguments to the binary, defaulting to --mcp --no-http
const args = process.argv.slice(2);
if (args.length === 0) {
  args.push('--mcp', '--no-http');
}

// Spawn the binary with inherited stdio for MCP communication
const child = spawn(BINARY_PATH, args, {
  stdio: 'inherit',
  env: process.env,
});

child.on('error', (err) => {
  console.error(`Failed to start contextd: ${err.message}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code || 0);
  }
});

// Forward signals to child process
['SIGINT', 'SIGTERM', 'SIGHUP'].forEach((signal) => {
  process.on(signal, () => {
    child.kill(signal);
  });
});
