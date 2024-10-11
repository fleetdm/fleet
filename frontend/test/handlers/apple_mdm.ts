import { rest } from "msw";

import { createMockVppInfo } from "__mocks__/appleMdm";
import { baseUrl } from "test/test-utils";

// eslint-disable-next-line import/prefer-default-export
export const defaultVppInfoHandler = rest.get(
  baseUrl("/vpp"),
  (req, res, context) => {
    return res(context.json(createMockVppInfo()));
  }
);

export const errorNoVppInfoHandler = rest.get(
  baseUrl("/vpp"),
  (req, res, context) => {
    return res(context.status(404));
  }
);
