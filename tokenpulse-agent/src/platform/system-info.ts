import { arch, hostname, platform } from 'node:os';

export function systemInfo(agentVersion: string, installationId: string) {
  return {
    deviceName: hostname(),
    platform: platform(),
    arch: arch(),
    agentVersion,
    installationId,
  };
}
