import React from "react";
import { render } from "@testing-library/react";

import { getErrorMessage } from "./helpers";

jest.mock("axios", () => {
  const actual = jest.requireActual("axios");
  return {
    ...actual,
    isAxiosError: () => true,
  };
});

describe("getErrorMessage", () => {
  it("returns message for pending install/uninstall error", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason:
                "Couldn't install. Host already has a pending install/uninstall for this installer.",
            },
          ],
        },
      },
    };

    expect(getErrorMessage(err)).toBe(
      "Couldn't add. Couldn't install. Host already has a pending install/uninstall for this installer."
    );
  });

  it("prepends a single 'Couldn't add.' to an action-neutral script validation reason", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason:
                "Script validation failed: Script is too large. It's limited to 500,000 characters (approximately 10,000 lines).",
            },
          ],
        },
      },
    };

    expect(getErrorMessage(err)).toBe(
      "Couldn't add. Script validation failed: Script is too large. It's limited to 500,000 characters (approximately 10,000 lines)."
    );
  });

  it("prepends 'Couldn't add.' to a corrupt tarball reason without doubling the verb", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason: "Uploaded file is not a valid .tar.gz archive.",
            },
          ],
        },
      },
    };

    const { container } = render(<>{getErrorMessage(err)}</>);
    expect(container.textContent).toContain(
      "Couldn't add. This is not a valid .tar.gz archive."
    );
    expect(container.textContent).not.toContain("Couldn't add. Couldn't add.");
  });
});
