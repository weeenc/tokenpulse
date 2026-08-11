import { execFile, spawn } from 'node:child_process';
import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { promisify } from 'node:util';
import type { AgentPaths } from '../platform/paths.js';

const execFileAsync = promisify(execFile);

export interface CredentialStore {
  saveDeviceToken(token: string): Promise<void>;
  getDeviceToken(): Promise<string | null>;
  deleteDeviceToken(): Promise<void>;
}

class MacOSCredentialStore implements CredentialStore {
  private readonly service = 'com.tokenpulse.agent';
  private readonly account = process.env.USER ?? 'tokenpulse';

  async saveDeviceToken(token: string): Promise<void> {
    await execFileAsync('security', [
      'add-generic-password',
      '-U',
      '-a',
      this.account,
      '-s',
      this.service,
      '-w',
      token,
    ]);
  }
  async getDeviceToken(): Promise<string | null> {
    try {
      const { stdout } = await execFileAsync('security', [
        'find-generic-password',
        '-a',
        this.account,
        '-s',
        this.service,
        '-w',
      ]);
      return stdout.trim() || null;
    } catch {
      return null;
    }
  }
  async deleteDeviceToken(): Promise<void> {
    try {
      await execFileAsync('security', [
        'delete-generic-password',
        '-a',
        this.account,
        '-s',
        this.service,
      ]);
    } catch {
      /* absent is already deleted */
    }
  }
}

class WindowsCredentialStore implements CredentialStore {
  constructor(private readonly paths: AgentPaths) {}

  async saveDeviceToken(token: string): Promise<void> {
    const script =
      '$v=[Console]::In.ReadToEnd();$b=[Text.Encoding]::UTF8.GetBytes($v);$e=[Security.Cryptography.ProtectedData]::Protect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Console]::Out.Write([Convert]::ToBase64String($e))';
    const encrypted = await powershell(script, token);
    await mkdir(this.paths.baseDir, { recursive: true });
    await writeFile(this.paths.credential, encrypted, { encoding: 'utf8', mode: 0o600 });
  }
  async getDeviceToken(): Promise<string | null> {
    try {
      const encrypted = await readFile(this.paths.credential, 'utf8');
      const script =
        '$v=[Console]::In.ReadToEnd();$b=[Convert]::FromBase64String($v);$d=[Security.Cryptography.ProtectedData]::Unprotect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Console]::Out.Write([Text.Encoding]::UTF8.GetString($d))';
      return (await powershell(script, encrypted)).trim() || null;
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ENOENT') return null;
      throw error;
    }
  }
  async deleteDeviceToken(): Promise<void> {
    await rm(this.paths.credential, { force: true });
  }
}

async function powershell(script: string, stdin: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', script], {
      stdio: ['pipe', 'pipe', 'pipe'],
      windowsHide: true,
    });
    let stdout = '';
    let stderr = '';
    child.stdout.setEncoding('utf8').on('data', (chunk: string) => {
      stdout += chunk;
    });
    child.stderr.setEncoding('utf8').on('data', (chunk: string) => {
      stderr += chunk;
    });
    child.on('error', reject);
    child.on('close', (code) =>
      code === 0 ? resolve(stdout) : reject(new Error(`DPAPI operation failed: ${stderr.trim()}`)),
    );
    child.stdin.end(stdin);
  });
}

class UnsupportedCredentialStore implements CredentialStore {
  async saveDeviceToken(): Promise<void> {
    throw new Error('Secure credential storage is supported on macOS and Windows only.');
  }
  async getDeviceToken(): Promise<string | null> {
    return null;
  }
  async deleteDeviceToken(): Promise<void> {}
}

export function createCredentialStore(
  paths: AgentPaths,
  platform: NodeJS.Platform = process.platform,
): CredentialStore {
  if (platform === 'darwin') return new MacOSCredentialStore();
  if (platform === 'win32') return new WindowsCredentialStore(paths);
  return new UnsupportedCredentialStore();
}

export async function credentialStoreAvailable(
  platform: NodeJS.Platform = process.platform,
): Promise<boolean> {
  try {
    if (platform === 'darwin') {
      await execFileAsync('security', ['list-keychains', '-d', 'user']);
      return true;
    }
    if (platform === 'win32') {
      const probe = `tokenpulse-health-${Date.now()}`;
      const protect =
        '$v=[Console]::In.ReadToEnd();$b=[Text.Encoding]::UTF8.GetBytes($v);$e=[Security.Cryptography.ProtectedData]::Protect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Console]::Out.Write([Convert]::ToBase64String($e))';
      const encrypted = await powershell(protect, probe);
      const unprotect =
        '$v=[Console]::In.ReadToEnd();$b=[Convert]::FromBase64String($v);$d=[Security.Cryptography.ProtectedData]::Unprotect($b,$null,[Security.Cryptography.DataProtectionScope]::CurrentUser);[Console]::Out.Write([Text.Encoding]::UTF8.GetString($d))';
      return (await powershell(unprotect, encrypted)) === probe;
    }
    return false;
  } catch {
    return false;
  }
}
