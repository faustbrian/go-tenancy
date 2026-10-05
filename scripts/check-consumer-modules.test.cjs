// Structural command-boundary proof only: stand-ins never compile Go or access services.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const root = fs.mkdtempSync(path.join(os.tmpdir(), 'tenancy-consumer-driver-test-'));
try {
  for (const name of ['foreign-cache', 'foreign-modules', 'foreign-tmp']) {
    fs.mkdirSync(path.join(root, name));
    fs.writeFileSync(path.join(root, name, 'preserved'), 'caller-owned sentinel');
  }
  const bin = path.join(root, 'bin');
  fs.mkdirSync(bin);
  fs.writeFileSync(path.join(bin, 'go'), `#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const args = process.argv.slice(2);
const imports = [];
function inspect(directory) {
  for (const entry of fs.readdirSync(directory, {withFileTypes:true})) {
    if (entry.name.startsWith('.')) continue;
    const item = path.join(directory, entry.name);
    if (entry.isDirectory()) inspect(item);
    else if (entry.name.endsWith('.go')) {
      const source = fs.readFileSync(item, 'utf8');
      for (const match of source.matchAll(/"(github[.]com[^"\\s]*)"/g)) {
        if (match[1].startsWith('github.com/faustbrian/go-tenancy')) imports.push(match[1]);
      }
    }
  }
}
if (args[0] === 'test') inspect(process.cwd());
fs.appendFileSync(process.env.DRIVER_CALLS, JSON.stringify({args, imports, cwd:process.cwd(),
  cache: fs.realpathSync(process.env.GOCACHE), modules:fs.realpathSync(process.env.GOMODCACHE),
  temporary:fs.realpathSync(process.env.GOTMPDIR),
  sumdb:process.env.GOSUMDB, proxy:process.env.GOPROXY, workspace:process.env.GOWORK,
  toolchain:process.env.GOTOOLCHAIN}) + String.fromCharCode(10));
if (args[0] === 'test') {
  const immutable = path.join(process.env.GOMODCACHE, 'readonly-module');
  fs.mkdirSync(immutable);
  fs.writeFileSync(path.join(immutable, 'go.mod'), 'inert cached module', {mode:0o444});
  fs.chmodSync(immutable, 0o555);
  process.exit(Number(process.env.DRIVER_STATUS));
}
`, { mode: 0o755 });
  for (const script of ['check-clean-consumer.sh', 'check-postgres-composition.sh',
    'check-redis-integration.sh', 'check-opensearch-integration.sh']) {
    for (const status of [0, 7]) {
      const temporary = path.join(root, `${script}-${status}`);
      fs.mkdirSync(temporary);
      const callsFile = path.join(root, `${script}-${status}.jsonl`);
      const result = spawnSync('bash', [path.join(__dirname, script)], {
        encoding:'utf8', env: { ...process.env, PATH:`${bin}:${process.env.PATH}`,
          TMPDIR:temporary, DRIVER_CALLS:callsFile, DRIVER_STATUS:String(status),
          POSTGRES_URL:'postgres://inert.invalid/fixture', REDIS_ADDR:'inert.invalid:6379',
          OPENSEARCH_URL:'http://inert.invalid', OPENSEARCH_EXPECTED_VERSION:'inert',
          GOCACHE:path.join(root, 'foreign-cache'), GOMODCACHE:path.join(root, 'foreign-modules'),
          GOTMPDIR:path.join(root, 'foreign-tmp'), GOSUMDB:'off', GOPROXY:'off', GOTOOLCHAIN:'auto' }
      });
      assert.equal(result.status, status, `${script} must preserve Go result`);
      const calls = fs.readFileSync(callsFile, 'utf8').trim().split('\n').map(JSON.parse);
      const edit = calls.find(call => call.args[0] === 'mod' && call.args[1] === 'edit');
      assert.ok(edit.args.includes('-require=github.com/faustbrian/go-tenancy/v2@v2.0.0'),
        `${script} must consume published Tenancy v2`);
      assert.ok(!edit.args.includes('-require=github.com/faustbrian/go-tenancy@v1.0.0'));
      if (script === 'check-clean-consumer.sh') {
        assert.ok(edit.args.includes('-require=github.com/faustbrian/go-cloudevents/adapters/golib@v1.0.0'),
          'intentional string-envelope v1 interoperability must remain');
      }
      const test = calls.find(call => call.args[0] === 'test');
      assert.ok(test.imports.length > 0);
      assert.ok(test.imports.every(item => item === 'github.com/faustbrian/go-tenancy/v2' ||
        item.startsWith('github.com/faustbrian/go-tenancy/v2/')), `${script} must select v2 imports`);
      for (const call of calls) {
        for (const cache of [call.cache, call.modules, call.temporary]) {
          assert.ok(cache && cache.startsWith(call.cwd + path.sep), 'every Go call must own disposable caches');
          assert.ok(!fs.existsSync(cache), 'owned Go caches must be removed');
        }
        assert.equal(call.sumdb, 'sum.golang.org');
        assert.equal(call.proxy, 'https://proxy.golang.org,direct');
        assert.equal(call.workspace, 'off');
        assert.equal(call.toolchain, 'local');
      }
      assert.deepEqual(fs.readdirSync(temporary), [], 'consumer cleanup must cover success and failure');
    }
  }
  for (const name of ['foreign-cache', 'foreign-modules', 'foreign-tmp']) {
    assert.equal(fs.readFileSync(path.join(root, name, 'preserved'), 'utf8'), 'caller-owned sentinel');
  }
  console.log('consumer module selection, public resolution policy, cache ownership and cleanup: PASS');
} finally {
  fs.rmSync(root, {recursive:true, force:true});
}
