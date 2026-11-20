#!/usr/bin/env node
// Compiles every page partial built from Markdown with Vue's own template compiler and fails if any of
// them contains a Vue expression (e.g. {{ foo }}). fleetdm.com compiles these pages as Vue 2 templates in
// the browser, so a stray expression throws at runtime and breaks the whole page for every visitor.
//
// Usage: node check-website-vue-templates.js <dir of built partials>
// Requires vue-template-compiler (same version as website/assets/dependencies/vue.js) to be resolvable.

const fs = require('fs');
const path = require('path');
const { compile } = require('vue-template-compiler');

const dir = process.argv[2];
if (!dir || !fs.existsSync(dir)) {
  console.error(`Built partials directory not found: ${dir}`);
  process.exit(2);
}

function walk(d) {
  return fs.readdirSync(d, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(d, e.name);
    return e.isDirectory() ? walk(p) : [p];
  });
}

// Pull each `_s(<expression>)` out of the generated render function so the error names what Vue would evaluate.
function interpolatedExpressions(render) {
  const found = [];
  let i = render.indexOf('_s(');
  while (i !== -1) {
    let depth = 0;
    let j = i + 2;
    for (; j < render.length; j++) {
      if (render[j] === '(') { depth++; }
      else if (render[j] === ')' && --depth === 0) { break; }
    }
    found.push(render.slice(i + 3, j));
    i = render.indexOf('_s(', j);
  }
  return found;
}

const files = walk(dir).filter((f) => f.endsWith('.ejs') || f.endsWith('.html'));
let failures = 0;
for (const file of files) {
  const { render } = compile(`<div>${fs.readFileSync(file, 'utf8')}</div>`);
  const expressions = interpolatedExpressions(render);
  if (expressions.length > 0) {
    failures++;
    console.error(`${path.relative(dir, file)}: Vue would evaluate ${expressions.map((e) => `{{ ${e} }}`).join(', ')}`);
  }
}

if (failures > 0) {
  console.error(`\n${failures} page(s) contain {{ … }} that Vue evaluates in the browser, which breaks the page. Put it in a code block or inline code (rendered with v-pre), or remove the double curly brackets.`);
  process.exit(1);
}
console.log(`Checked ${files.length} built pages: no Vue expressions found.`);
process.exit(0);// « vue-template-compiler keeps the event loop alive, so exit explicitly.
