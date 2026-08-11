export type DeviceChoice = 'new' | number;

export function approvalPayload(userCode: string, choice: DeviceChoice) {
  return {
    userCode,
    ...(choice === 'new' ? {} : { targetDeviceId: choice }),
  };
}

export function platformName(value: string): string {
  if (value === 'darwin') return 'macOS';
  if (value === 'win32') return 'Windows';
  return value;
}
