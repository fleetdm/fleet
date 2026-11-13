import { render } from "@testing-library/react";
import React from "react";

import { ISoftwarePackage } from "interfaces/software";

import { getErrorMessage, isNonSideEffectOnlyChange } from "./helpers";

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

// Regression: the Edit software modal's "cancel pending installs" warning
// used to always fire because the diff carried extra form-only fields
// (categories, automaticInstall, the installer reference seeded as `software`)
// that current-state didn't. The modal now routes the diff through this
// helper, which recognizes a self-service-only or categories-only edit as
// non-side-effect and skips the warning. Mirrors the backend's
// `fieldsShouldSideEffect` map in ee/server/service/software_installers.go.
describe("isNonSideEffectOnlyChange", () => {
  it("is false for an empty diff so an unchanged save doesn't skip the modal by accident", () => {
    expect(isNonSideEffectOnlyChange({})).toBe(false);
  });

  it("is true when only self-service changed", () => {
    expect(isNonSideEffectOnlyChange({ selfService: true })).toBe(true);
  });

  it("is true when only categories changed", () => {
    expect(isNonSideEffectOnlyChange({ categories: ["Productivity"] })).toBe(
      true
    );
  });

  it("is true when both self-service and categories changed and nothing else", () => {
    expect(
      isNonSideEffectOnlyChange({
        selfService: true,
        categories: ["Productivity"],
      })
    ).toBe(true);
  });

  it("is false when a side-effect field like install script is also in the diff", () => {
    expect(
      isNonSideEffectOnlyChange({
        selfService: true,
        installScript: "echo hi",
      })
    ).toBe(false);
  });

  it("is false when the installer file is in the diff", () => {
    expect(isNonSideEffectOnlyChange({ software: {} })).toBe(false);
  });

  it("is false when label targeting is in the diff", () => {
    expect(
      isNonSideEffectOnlyChange({ labelTargets: { "label-1": true } })
    ).toBe(false);
  });
});
