#!/usr/bin/env node
// Locus hook: Stop
// Writes a session_end event to %LOCALAPPDATA%\Locus\events.jsonl

const fs = require('fs');
const path = require('path');

const eventsDir = path.join(process.env.LOCALAPPDATA || process.env.APPDATA || '', 'Locus');
const eventsPath = path.join(eventsDir, 'events.jsonl');

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', () => {
  try {
    const data = JSON.parse(input);
    const event = {
      type: 'session_end',
      session_id: data.session_id || '',
      ts: Math.floor(Date.now() / 1000),
    };
    fs.mkdirSync(eventsDir, { recursive: true });
    fs.appendFileSync(eventsPath, JSON.stringify(event) + '\n');
  } catch (_) {
    // Silently fail; never block Claude.
  }
  process.exit(0);
});
