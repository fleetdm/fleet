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

  it("returns message for script validation error", () => {
    const err = {
      response: {
        status: 400,
        data: {
          errors: [
            {
              name: "Error",
              reason:
                "Couldn't add. Script validation failed: Script is too large. It's limited to 500,000 characters (approximately 10,000 lines).",
            },
          ],
        },
      },
    };

    expect(getErrorMessage(err)).toBe(
      "Couldn't add. Script validation failed: Script is too large. It's limited to 500,000 characters (approximately 10,000 lines)."
    );
  });
});
