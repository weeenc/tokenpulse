import { describe, expect, it } from 'vitest';
import { approvalPayload, platformName } from '../src/utils/device-auth.js';

describe('device authorization helpers', () => {
  it('does not send a target for a new device', () => {
    expect(approvalPayload('ABCD-EFGH', 'new')).toEqual({ userCode: 'ABCD-EFGH' });
  });

  it('sends the selected owned device for reconnect', () => {
    expect(approvalPayload('ABCD-EFGH', 42)).toEqual({
      userCode: 'ABCD-EFGH',
      targetDeviceId: 42,
    });
  });

  it('normalizes platform labels', () => {
    expect(platformName('darwin')).toBe('macOS');
    expect(platformName('win32')).toBe('Windows');
  });
});
