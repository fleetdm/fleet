import React from "react";
import { render } from "@testing-library/react";

import { ISoftwarePackage } from "interfaces/software";

import { getErrorMessage } from "./helpers";

jest.mock("axios", () => {
  const actual = jest.requireActual("axios");
  return {
    ...actual,
    isAxiosError: () => true,
  };
});

const software = {
  name: "Zoom.pkg",
  display_name: "Zoom",
} as ISoftwarePackage;

describe("getErrorMessage", () => {
  it("prepends 'Couldn't edit software.' to an action-neutral validator reason", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason:
                'Script validation failed: Python scripts must start with a python shebang (for example, "#!/usr/bin/env python3").',
            },
          ],
        },
      },
    };

    expect(getErrorMessage(err, software)).toBe(
      'Couldn\'t edit software. Script validation failed: Python scripts must start with a python shebang (for example, "#!/usr/bin/env python3").'
    );
  });

  it("does not double the verb when the reason has no recognized special case", () => {
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

    expect(getErrorMessage(err, software)).toBe(
      "Couldn't edit software. Uploaded file is not a valid .tar.gz archive."
    );
  });

  it("normalizes a backend reason that carries its own verb to the product wording (no doubling)", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason:
                "Couldn't edit. Install script is required for .exe packages.",
            },
          ],
        },
      },
    };

    expect(getErrorMessage(err, software)).toBe(
      "Couldn't edit software. Install script is required for .exe packages."
    );
  });

  it("returns the default message when there is no reason", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [],
        },
      },
    };

    expect(getErrorMessage(err, software)).toBe(
      "Couldn't edit software. Please try again."
    );
  });

  it("bolds the software name for the different-file-type reshape", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason: "The selected package is for a different file type.",
            },
          ],
        },
      },
    };

    const { container } = render(<>{getErrorMessage(err, software)}</>);
    expect(container.textContent).toBe(
      "Couldn't edit Zoom. The selected package is for a different file type."
    );
  });
});
