import { execFile } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { promisify } from 'node:util';
import { agentPaths } from '../platform/paths.js';

const execFileAsync = promisify(execFile);
const label = 'com.tokenusage.agent';
const windowsLauncherFilename = 'autosubmit.vbs';

export function intervalSeconds(value: string): number {
  const match = /^(\d+)(m|h)$/.exec(value);
  if (!match) throw new Error('Interval must use minutes or hours, for example 30m or 1h.');
  const amount = Number(match[1]);
  if (amount < 1) throw new Error('Interval must be positive.');
  return amount * (match[2] === 'h' ? 3600 : 60);
}

export async function enableAutosubmit(interval: string): Promise<void> {
  const seconds = intervalSeconds(interval);
  if (process.platform === 'darwin') return enableMacOS(seconds);
  if (process.platform === 'win32') return enableWindows(seconds);
  throw new Error('Autosubmit is supported on macOS and Windows only.');
}

export async function disableAutosubmit(): Promise<void> {
  if (process.platform === 'darwin') {
    const path = plistPath();
    try {
      await execFileAsync('launchctl', ['bootout', `gui/${process.getuid?.() ?? 0}`, path]);
    } catch {
      /* task may not be loaded */
    }
    await rm(path, { force: true });
    return;
  }
  if (process.platform === 'win32') {
    try {
      await execFileAsync('schtasks', ['/Delete', '/TN', 'TokenPulse Agent Sync', '/F']);
    } catch {
      /* task may not exist */
    }
    await rm(windowsLauncherPath(), { force: true });
    return;
  }
  throw new Error('Autosubmit is supported on macOS and Windows only.');
}

export async function autosubmitStatus(): Promise<boolean> {
  if (process.platform === 'darwin') {
    try {
      await execFileAsync('launchctl', ['print', `gui/${process.getuid?.() ?? 0}/${label}`]);
      return true;
    } catch {
      return false;
    }
  }
  if (process.platform === 'win32') {
    try {
      await execFileAsync('schtasks', ['/Query', '/TN', 'TokenPulse Agent Sync']);
      return true;
    } catch {
      return false;
    }
  }
  return false;
}

async function enableMacOS(seconds: number): Promise<void> {
  const path = plistPath();
  await mkdir(join(homedir(), 'Library', 'LaunchAgents'), { recursive: true });
  const cliPath = process.argv[1];
  if (!cliPath) throw new Error('Unable to determine TokenPulse CLI path.');
  const xml = launchAgentXml(process.execPath, cliPath, seconds);
  await writeFile(path, xml, 'utf8');
  try {
    await execFileAsync('launchctl', ['bootout', `gui/${process.getuid?.() ?? 0}`, path]);
  } catch {
    /* replace existing */
  }
  await execFileAsync('launchctl', ['bootstrap', `gui/${process.getuid?.() ?? 0}`, path]);
}

async function enableWindows(seconds: number): Promise<void> {
  const cliPath = process.argv[1];
  if (!cliPath) throw new Error('Unable to determine TokenPulse CLI path.');
  const minutes = Math.max(1, Math.ceil(seconds / 60));
  const launcherPath = windowsLauncherPath();
  await mkdir(agentPaths('win32').baseDir, { recursive: true });
  await writeFile(
    launcherPath,
    `\uFEFF${windowsLauncherScript(process.execPath, cliPath)}`,
    'utf16le',
  );
  const wscriptPath = join(process.env.SystemRoot ?? 'C:\\Windows', 'System32', 'wscript.exe');
  const command = windowsLauncherCommand(wscriptPath, launcherPath);
  await execFileAsync('schtasks', [
    '/Create',
    '/TN',
    'TokenPulse Agent Sync',
    '/SC',
    'MINUTE',
    '/MO',
    String(minutes),
    '/TR',
    command,
    '/F',
  ]);
}

function plistPath(): string {
  return join(homedir(), 'Library', 'LaunchAgents', `${label}.plist`);
}

export function windowsTaskCommand(nodePath: string, cliPath: string): string {
  return `"${nodePath.replaceAll('"', '\\"')}" "${cliPath.replaceAll('"', '\\"')}" sync`;
}

export function windowsLauncherScript(nodePath: string, cliPath: string): string {
  const command = windowsTaskCommand(nodePath, cliPath).replaceAll('"', '""');
  return [
    'Option Explicit',
    'Dim shell, exitCode',
    'Set shell = CreateObject("WScript.Shell")',
    `exitCode = shell.Run("${command}", 0, True)`,
    'WScript.Quit exitCode',
    '',
  ].join('\r\n');
}

export function windowsLauncherCommand(wscriptPath: string, launcherPath: string): string {
  return `"${wscriptPath.replaceAll('"', '\\"')}" //B //NoLogo "${launcherPath.replaceAll('"', '\\"')}"`;
}

function windowsLauncherPath(): string {
  return join(agentPaths('win32').baseDir, windowsLauncherFilename);
}

export function launchAgentXml(nodePath: string, cliPath: string, seconds: number): string {
  return `<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0"><dict><key>Label</key><string>${label}</string><key>ProgramArguments</key><array><string>${escapeXml(nodePath)}</string><string>${escapeXml(cliPath)}</string><string>sync</string></array><key>StartInterval</key><integer>${seconds}</integer><key>RunAtLoad</key><true/></dict></plist>\n`;
}

function escapeXml(value: string): string {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
}
