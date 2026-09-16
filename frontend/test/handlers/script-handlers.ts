import { http, HttpResponse } from "msw";

import { createMockScript } from "__mocks__/scriptMock";
import { IScript } from "interfaces/script";
import { baseUrl } from "test/test-utils";

// not supported for all teams
const getTeamScriptsHandler = (
  teamId: number,
  overrides: Partial<IScript>[]
) => {
  const scripts = overrides.map((scriptOverride) =>
    createMockScript(scriptOverride)
  );
  return http.get(baseUrl(`/scripts?fleet_id=${teamId}`), () =>
    HttpResponse.json({
      scripts,
    })
  );
};

export default getTeamScriptsHandler;
