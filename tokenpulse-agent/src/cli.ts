#!/usr/bin/env node
import { readFile } from 'node:fs/promises';
import { Command } from 'commander';
import { agentPaths } from './platform/paths.js';
import { ConfigStore } from './storage/config.js';
import { createCredentialStore } from './auth/credential-store.js';
import { login } from './auth/device-auth.js';
import { collect, status, sync } from './app.js';
import { doctor } from './commands/doctor.js';
import { autosubmitStatus, disableAutosubmit, enableAutosubmit } from './scheduler/index.js';

type PackageJson = { version: string };
const packageJson = JSON.parse(
  await readFile(new URL('../package.json', import.meta.url), 'utf8'),
) as PackageJson;
const paths = agentPaths();
const configStore = new ConfigStore(paths);
const credentials = createCredentialStore(paths);
const program = new Command();

program
  .name('tokenpulse')
  .description('Collect and synchronize AI coding token usage metadata')
  .version(packageJson.version)
  .option('--server <url>', 'TokenPulse server URL');
program
  .command('login')
  .description('Connect this installation to a TokenPulse account')
  .option('--no-browser', 'Do not open the browser automatically')
  .action(async (options: { browser: boolean }) => {
    const server = program.opts<{ server?: string }>().server;
    await login(configStore, credentials, packageJson.version, {
      ...(server ? { server } : {}),
      browser: options.browser,
    });
  });
program
  .command('logout')
  .description('Delete local device authentication')
  .action(async () => {
    await credentials.deleteDeviceToken();
    console.log('✓ Logged out locally.');
  });
program
  .command('status')
  .description('Show authentication and synchronization status')
  .action(() =>
    status(
      { configStore, credentials },
      packageJson.version,
      program.opts<{ server?: string }>().server,
    ),
  );
program
  .command('collect')
  .description('Collect usage metadata without uploading')
  .option('--verbose')
  .action(async (options: { verbose?: boolean }) => {
    const result = await collect(configStore, options.verbose);
    console.log(`✓ Collected ${result.inserted} new events (${result.found} scanned).`);
  });
program
  .command('sync')
  .description('Collect and upload usage metadata')
  .option('--verbose')
  .action(async (options: { verbose?: boolean }) => {
    const result = await sync(
      { configStore, credentials },
      options.verbose,
      program.opts<{ server?: string }>().server,
    );
    console.log(
      `✓ Sync complete: ${result.collected} collected, ${result.uploaded} uploaded, ${result.duplicated} already present.`,
    );
  });
program
  .command('doctor')
  .description('Check the local TokenPulse installation')
  .action(() => doctor(configStore, credentials, program.opts<{ server?: string }>().server));
const autosubmit = program.command('autosubmit').description('Manage scheduled synchronization');
autosubmit
  .command('enable')
  .option('--interval <duration>', 'Sync interval', '1h')
  .action(async (options: { interval: string }) => {
    await enableAutosubmit(options.interval);
    console.log(`✓ Autosubmit enabled (${options.interval}).`);
  });
autosubmit.command('disable').action(async () => {
  await disableAutosubmit();
  console.log('✓ Autosubmit disabled.');
});
autosubmit.command('status').action(async () => {
  console.log((await autosubmitStatus()) ? '✓ Autosubmit enabled' : '✗ Autosubmit disabled');
});
autosubmit.command('run').action(async () => {
  const result = await sync(
    { configStore, credentials },
    false,
    program.opts<{ server?: string }>().server,
  );
  console.log(`✓ Autosubmit test complete: ${result.uploaded} uploaded.`);
});

program.parseAsync().catch((error: unknown) => {
  console.error(`✗ ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
});
