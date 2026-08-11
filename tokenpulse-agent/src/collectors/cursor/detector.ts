import { access } from 'node:fs/promises';

export class CursorSourceDetector {
  constructor(private readonly databasePath: string) {}

  async detect(): Promise<boolean> {
    try {
      await access(this.databasePath);
      return true;
    } catch {
      return false;
    }
  }
}
