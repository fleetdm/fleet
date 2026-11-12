import { http, HttpResponse } from "msw";

import { IDiskEncryptionSummaryResponse } from "services/entities/disk_encryption";
import { baseUrl } from "test/test-utils";

const diskEncryptionUrl = baseUrl("/disk_encryption");

const DEFAULT_DISK_ENCRYPTION_SUMMARY: IDiskEncryptionSummaryResponse = {
  verified: { macos: 1, windows: 2, linux: 3 },
  verifying: { macos: 0, windows: 0, linux: 0 },
  action_required: { macos: 0, windows: 0, linux: 0 },
  enforcing: { macos: 0, windows: 0, linux: 0 },
  failed: { macos: 0, windows: 0, linux: 0 },
  removing_enforcement: { macos: 0, windows: 0, linux: 0 },
};

export const createGetDiskEncryptionSummaryHandler = (
  overrides?: Partial<IDiskEncryptionSummaryResponse>
) => {
  return http.get(diskEncryptionUrl, () => {
    return HttpResponse.json({
      ...DEFAULT_DISK_ENCRYPTION_SUMMARY,
      ...overrides,
    });
  });
};

export const createUpdateDiskEncryptionHandler = (
  onRequest?: (body: Record<string, unknown>) => void
) => {
  return http.post(diskEncryptionUrl, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    onRequest?.(body);
    return HttpResponse.json({});
  });
};
