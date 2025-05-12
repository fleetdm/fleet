import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import { IScript } from "interfaces/script";
import { createMockScript } from "__mocks__/scriptMock";

// not supported for all teams
const getTeamScriptsHandler = (
  teamId: number,
  overrides: Partial<IScript>[]
) => {
  const scripts = overrides.map((scriptOverride) =>
    createMockScript(scriptOverride)
  );
  return http.get(baseUrl(`/scripts?team_id=${teamId}`), () =>
    HttpResponse.json({
      scripts,
    })
  );
};

export default getTeamScriptsHandler;
