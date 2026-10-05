// Inert command-boundary test: no Go compilation or real scanner execution.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const root = fs.mkdtempSync(path.join(os.tmpdir(), 'tenancy-scanner-test-'));
try {
  const bin = path.join(root, 'bin');
  fs.mkdirSync(bin);
  fs.writeFileSync(path.join(bin, 'go'), `#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const args = process.argv.slice(2);
fs.appendFileSync(process.env.GO_CALLS, JSON.stringify({args, cwd:process.cwd()}) + '\\n');
if (args[0] === 'install') {
  if (args[1] === 'github.com/securego/gosec/v2/cmd/gosec@v2.29.0') {
    fs.writeFileSync(path.join(process.env.GOBIN, 'gosec'), \`#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const args = process.argv.slice(2);
fs.writeFileSync(process.env.SCANNER_CALL, JSON.stringify({
  args, version: 'v2.29.0', cwd: process.cwd(), module: fs.readFileSync('go.mod', 'utf8'),
  rootSources: fs.readdirSync('.').filter(name => name.endsWith('.go')),
  exportsPrepared: fs.readFileSync(process.env.GO_CALLS, 'utf8').trim().split(String.fromCharCode(10))
    .map(line => JSON.parse(line)).some(call => call.cwd === process.cwd() &&
      JSON.stringify(call.args) === JSON.stringify(['list','-deps','-export','./...'])),
  packages: ['adapter','consumer','metrics'].filter(name =>
    fs.existsSync(path.join('analyzerfixture', name, name + '.go')))
}));
const report = args.find(arg => arg.startsWith('-out='));
if (report) fs.writeFileSync(report.slice(5), JSON.stringify({Issues:[], 'Golang errors':{}}));
console.log('private scanner output sentinel');
process.exit(Number(process.env.SCANNER_STATUS || 0));
\`, { mode: 0o755 });
    process.exit(0);
  }
  fs.writeFileSync(path.join(process.env.GOBIN, 'golib-analysis'), \`#!/usr/bin/env node
const args = process.argv.slice(2);
if (args[0] === 'validate-config') process.exit(0);
if (args.at(-1).endsWith('/consumer')) {
  const diagnostics = [...Array(5)].map(() => ({rule:'api/forbidden-call'}));
  diagnostics.push({rule:'context/no-background'}, {rule:'observability/high-cardinality-label'});
  console.log(JSON.stringify({diagnostics}));
  process.exit(1);
}
console.log(JSON.stringify({diagnostics:[]}));
\`, { mode: 0o755 });
} else if (args[0] === 'mod' && args[1] === 'init') {
  fs.writeFileSync('go.mod', 'module ' + args[2] + '\\n');
} else if (args[0] === 'mod' && args[1] === 'edit') {
  fs.appendFileSync('go.mod', args.slice(2).join('\\n'));
}
`, { mode: 0o755 });
  for (const scannerStatus of [0, 7]) {
    const temporary = path.join(root, `run-${scannerStatus}`);
    fs.mkdirSync(temporary);
    const call = path.join(root, `call-${scannerStatus}.json`);
    const goCalls = path.join(root, `go-calls-${scannerStatus}.jsonl`);
    const result = spawnSync('bash', [path.join(__dirname, 'check-analyzers.sh'), '--security'], {
      encoding: 'utf8', env: { ...process.env, PATH: `${bin}:${process.env.PATH}`,
        TMPDIR: temporary, SCANNER_CALL: call, GO_CALLS: goCalls,
        SCANNER_STATUS: String(scannerStatus) }
    });
    assert.ok(fs.existsSync(call), 'prepared fixture security scan must run');
    const invocation = JSON.parse(fs.readFileSync(call, 'utf8'));
    assert.equal(invocation.version, 'v2.29.0');
    assert.equal(invocation.args.at(-1), './...');
    assert.ok(invocation.args.includes('-nosec-require-rules'));
    assert.ok(invocation.args.includes('-nosec-require-justification'));
    assert.deepEqual(invocation.packages, ['adapter','consumer','metrics']);
    assert.ok(invocation.rootSources.length > 0, 'prepared root package must be present');
    assert.ok(invocation.exportsPrepared, 'fixture export caches must be prepared before scanner');
    for (const dependency of ['audit','cache','queue','telemetry','workflow']) {
      assert.ok(invocation.module.includes(`github.com/faustbrian/go-${dependency}@v1.0.0`));
    }
    assert.equal(result.status, scannerStatus, 'scanner refusal must propagate');
    assert.ok(!(result.stdout + result.stderr).includes('private scanner output sentinel'));
    assert.deepEqual(fs.readdirSync(temporary), [], 'prepared fixture and private reports must be removed');
  }
  const temporary = path.join(root, 'analyzer-only');
  fs.mkdirSync(temporary);
  const call = path.join(root, 'analyzer-only.json');
  const goCalls = path.join(root, 'analyzer-only-go.jsonl');
  const ordinary = spawnSync('bash', [path.join(__dirname, 'check-analyzers.sh')], {
    encoding: 'utf8', env: { ...process.env, PATH: `${bin}:${process.env.PATH}`,
      TMPDIR: temporary, SCANNER_CALL: call, GO_CALLS: goCalls }
  });
  assert.equal(ordinary.status, 0, 'ordinary analyzer mode must still pass');
  assert.ok(!fs.existsSync(call), 'ordinary analyzer mode must not run scanner');
  const calls = fs.readFileSync(goCalls, 'utf8').trim().split('\n').map(line => JSON.parse(line));
  assert.ok(!calls.some(call => call.args.some(arg => arg.includes('securego/gosec'))),
    'ordinary analyzer mode must not install scanner');
  assert.deepEqual(fs.readdirSync(temporary), []);
  const invalidCalls = path.join(root, 'invalid-go.jsonl');
  const invalid = spawnSync('bash', [path.join(__dirname, 'check-analyzers.sh'), '--invalid'], {
    encoding: 'utf8', env: { ...process.env, PATH: `${bin}:${process.env.PATH}`,
      TMPDIR: temporary, GO_CALLS: invalidCalls }
  });
  assert.equal(invalid.status, 2, 'unsupported modes must fail usage validation');
  assert.ok(!fs.existsSync(invalidCalls), 'invalid mode must fail before any Go command');
  assert.deepEqual(fs.readdirSync(temporary), []);
  console.log('prepared fixture scan invocation, refusal, privacy and cleanup: PASS');
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
