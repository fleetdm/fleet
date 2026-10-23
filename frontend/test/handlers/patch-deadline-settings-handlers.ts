import { http, HttpResponse } from "msw";

import {
  IPatchDeadlineApp,
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import { baseUrl } from "test/test-utils";

const settingsUrl = baseUrl("/software/patch_deadline_settings");
const appsUrl = baseUrl("/software/patch_deadline_settings/apps");

export const getPatchDeadlineSettingsHandler = (
  overrides?: Partial<IPatchDeadlineSettings>
) =>
  http.get(settingsUrl, ({ request }) => {
    const url = new URL(request.url);
    const fleetId = Number(url.searchParams.get("fleet_id") ?? 0);
    const body: IPatchDeadlineSettings = {
      fleet_id: fleetId,
      days: ["tuesday"] as Weekday[],
      time: "17:00",
      ...overrides,
    };
    return HttpResponse.json(body);
  });

export const updatePatchDeadlineSettingsHandler = http.patch(
  settingsUrl,
  async ({ request }) => {
    const body = (await request.json()) as IPatchDeadlineSettings;
    return HttpResponse.json(body);
  }
);

export const updatePatchDeadlineSettingsBadRequestHandler = http.patch(
  settingsUrl,
  () =>
    HttpResponse.json(
      {
        message: "Validation error",
        errors: [{ name: "time", reason: "bad" }],
      },
      { status: 400 }
    )
);

// Factory for simulating field-scoped 400 validation errors from the backend.
// Pass per-field reasons to drive the inline-error assertion.
export const updatePatchDeadlineSettingsFieldErrorHandler = (
  fieldErrors: { name: "days" | "time"; reason: string }[]
) =>
  http.patch(settingsUrl, () =>
    HttpResponse.json(
      { message: "Validation error", errors: fieldErrors },
      { status: 400 }
    )
  );

// Hangs the response so tests can assert the Save button's in-flight state.
// Returning a never-resolving promise keeps the mutation stuck in `isLoading`.
export const updatePatchDeadlineSettingsPendingHandler = http.patch(
  settingsUrl,
  () =>
    new Promise<never>(() => {
      /* intentionally never resolves */
    })
);

export const updatePatchDeadlineSettingsServerErrorHandler = http.patch(
  settingsUrl,
  () => HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);

export const getPatchDeadlineAppsHandler = (
  apps: Partial<IPatchDeadlineApp>[] = []
) =>
  http.get(appsUrl, () =>
    HttpResponse.json({
      apps: apps.map((a, i) => ({
        id: a.id ?? i + 1,
        name: a.name ?? `App ${i + 1}`,
        icon_url: a.icon_url ?? null,
        source: a.source ?? "apps",
      })),
      count: apps.length,
    })
  );

export const emptyPatchDeadlineAppsHandler = getPatchDeadlineAppsHandler([]);

export const getPatchDeadlineAppsErrorHandler = http.get(appsUrl, () =>
  HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);
