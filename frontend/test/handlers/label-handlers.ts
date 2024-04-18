import { rest } from "msw";

import { baseUrl } from "test/test-utils";
import { createMockLabel } from "__mocks__/labelsMock";
import { ILabel } from "interfaces/label";

// eslint-disable-next-line import/prefer-default-export
export const getLabelHandler = (overrides: Partial<ILabel>) =>
  rest.get(baseUrl("/labels/:id"), (req, res, context) => {
    return res(
      context.json({
        label: createMockLabel({ ...overrides }),
      })
    );
  });
