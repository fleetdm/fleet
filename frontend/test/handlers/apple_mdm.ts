import { rest } from "msw";

import { createMockVppInfo, createMockScepInfo } from "__mocks__/appleMdm";
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

// eslint-disable-next-line import/prefer-default-export
export const defaultScepInfoHandler = rest.get(
  baseUrl("/scep"),
  (req, res, context) => {
    return res(context.json(createMockScepInfo()));
  }
);

export const errorNoScepInfoHandler = rest.get(
  baseUrl("/scep"),
  (req, res, context) => {
    return res(context.status(404));
  }
);
