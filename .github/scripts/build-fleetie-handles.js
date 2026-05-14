#!/usr/bin/env node
// Builds a list of GitHub usernames belonging to current or former Fleeties.
//
// Sources:
//   1. Current: GET /orgs/fleetdm/members (requires READ_ORG_TOKEN with read:org).
//      If READ_ORG_TOKEN is unset, this step is skipped and the script runs in handbook-only mode.
//   2. Former: handles that have ever appeared in handbook files as `[@x](https://github.com/x)` links.
//      Collected by walking `git log -p --follow` over each handbook file.
//
// Output: newline-delimited lowercased handles, sorted, deduped, to stdout or to FLEETIE_HANDLES_OUT.

'use strict';

const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const https = require('node:https');
const path = require('node:path');

const HANDBOOK_FILES = [
  'handbook/company/product-groups.md',
  'handbook/company/go-to-market-operations.md',
  'handbook/company/communications.md',
  'handbook/ceo/README.md',
  'handbook/customer-success/README.md',
  'handbook/engineering/README.md',
  'handbook/finance/README.md',
  'handbook/it/README.md',
  'handbook/marketing/README.md',
  'handbook/marketing/marketing-responsibilities.md',
  'handbook/people/README.md',
  'handbook/product-design/README.md',
  'handbook/sales/README.md',
];

// Handles that the regex captures but that are not personal accounts.
const DENYLIST = new Set([
  'fleetdm',
  'fleetdm-bot',
  'todo',
  'orgs',
  'issues',
  'pull',
  'pulls',
  'user-attachments',
  'apps',
  'features',
  'about',
  'sponsors',
  'marketplace',
  'enterprise',
  'topics',
  'collections',
  'login',
  'logout',
  'settings',
  'notifications',
  'security',
  'pricing',
  'contact',
  'open-source',
  'readme',
  'search',
  'explore',
  'trending',
  'mobile',
  'team',
  'customer-stories',
  'github',
  'organizations',
  'new',
  'edit',
]);

// Match `https://github.com/<handle>)` exactly. The trailing `)` ensures we only capture handles inside
// `[@x](https://github.com/x)` markdown links and not URL path segments like `/orgs/...`.
// GitHub handles: 1-39 chars, alphanumeric or hyphen, must not start with hyphen.
const HANDLE_RE = /github\.com\/([A-Za-z0-9][A-Za-z0-9-]{0,38})\)/g;

function gitRoot() {
  return execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
}

function extractHandles(text, into) {
  HANDLE_RE.lastIndex = 0;
  let match;
  while ((match = HANDLE_RE.exec(text)) !== null) {
    const handle = match[1].toLowerCase();
    if (handle.endsWith('-')) continue;
    if (DENYLIST.has(handle)) continue;
    into.add(handle);
  }
}

function collectFromHead(file, into) {
  if (!fs.existsSync(file)) return;
  extractHandles(fs.readFileSync(file, 'utf8'), into);
}

function collectFromGitHistory(file, into) {
  let stdout;
  try {
    stdout = execFileSync(
      'git',
      ['log', '-p', '--follow', '--no-color', '--pretty=format:', '--', file],
      { encoding: 'utf8', maxBuffer: 256 * 1024 * 1024 },
    );
  } catch (err) {
    process.stderr.write(`warn: git log failed for ${file}: ${err.message}\n`);
    return;
  }
  extractHandles(stdout, into);
}

function parseLinkNext(linkHeader) {
  if (!linkHeader) return null;
  for (const part of linkHeader.split(',')) {
    const match = part.match(/^\s*<([^>]+)>;\s*rel="next"\s*$/);
    if (match) return match[1];
  }
  return null;
}

function httpGet(url, token) {
  return new Promise((resolve, reject) => {
    const opts = {
      headers: {
        'User-Agent': 'fleetdm-build-fleetie-handles',
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        Authorization: `Bearer ${token}`,
      },
    };
    https
      .get(url, opts, (res) => {
        let body = '';
        res.setEncoding('utf8');
        res.on('data', (chunk) => (body += chunk));
        res.on('end', () => {
          if (res.statusCode && res.statusCode >= 200 && res.statusCode < 300) {
            resolve({ body, linkNext: parseLinkNext(res.headers.link || '') });
          } else {
            reject(new Error(`HTTP ${res.statusCode} from ${url}: ${body.slice(0, 200)}`));
          }
        });
      })
      .on('error', reject);
  });
}

async function fetchOrgMembers(token, into) {
  let url = 'https://api.github.com/orgs/fleetdm/members?per_page=100';
  while (url) {
    const { body, linkNext } = await httpGet(url, token);
    const arr = JSON.parse(body);
    if (!Array.isArray(arr)) {
      throw new Error(`org members API returned non-array: ${body.slice(0, 200)}`);
    }
    for (const member of arr) {
      if (member && typeof member.login === 'string') {
        into.add(member.login.toLowerCase());
      }
    }
    url = linkNext;
  }
}

async function main() {
  process.chdir(gitRoot());

  const handles = new Set();
  const token = process.env.READ_ORG_TOKEN || '';

  if (token) {
    try {
      await fetchOrgMembers(token, handles);
      process.stderr.write(`info: fetched fleetdm org members; running total ${handles.size}\n`);
    } catch (err) {
      process.stderr.write(`warn: org members fetch failed (${err.message}); using handbook-only\n`);
    }
  } else {
    process.stderr.write('info: READ_ORG_TOKEN not set; using handbook-only sources\n');
  }

  for (const file of HANDBOOK_FILES) {
    collectFromHead(file, handles);
    collectFromGitHistory(file, handles);
  }

  // Re-apply the denylist after the union so we cannot leak `fleetdm` or `todo` even if a future source
  // emitted them in non-lowercased form.
  for (const denied of DENYLIST) {
    handles.delete(denied);
  }

  const out = [...handles].sort().join('\n') + '\n';
  const outPath = process.env.FLEETIE_HANDLES_OUT;
  if (outPath) {
    fs.mkdirSync(path.dirname(outPath), { recursive: true });
    fs.writeFileSync(outPath, out);
    process.stderr.write(`info: wrote ${handles.size} handles to ${outPath}\n`);
  } else {
    process.stdout.write(out);
  }
}

main().catch((err) => {
  process.stderr.write(`error: ${err.stack || err.message}\n`);
  process.exit(1);
});
