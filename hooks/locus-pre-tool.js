#!/usr/bin/env node
// Locus hook: PreToolUse
// Writes a tool_use event to %LOCALAPPDATA%\Locus\events.jsonl

const fs = require('fs');
const path = require('path');

const eventsDir = path.join(process.env.LOCALAPPDATA || process.env.APPDATA || '', 'Locus');
const eventsPath = path.join(eventsDir, 'events.jsonl');

// Tools that Locus tracks. Others are silently skipped to reduce file growth.
const ITEM_TOOLS = new Set(['Edit', 'Write', 'NotebookEdit', 'Bash']);

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', () => {
  try {
    const data = JSON.parse(input);
    const tool = data.tool_name || '';
    if (!ITEM_TOOLS.has(tool)) {
      process.exit(0);
      return;
    }
    const inp = data.tool_input || {};
    // Extract the most relevant "target" field for the tool type.
    const target = inp.file_path || inp.command || inp.pattern || '';
    const event = {
      type: 'tool_use',
      session_id: data.session_id || '',
      tool: tool,
      target: target,
      ts: Math.floor(Date.now() / 1000),
    };
    fs.mkdirSync(eventsDir, { recursive: true });
    fs.appendFileSync(eventsPath, JSON.stringify(event) + '\n');
  } catch (_) {
    // Silently fail; never block Claude.
  }
  process.exit(0);
});
