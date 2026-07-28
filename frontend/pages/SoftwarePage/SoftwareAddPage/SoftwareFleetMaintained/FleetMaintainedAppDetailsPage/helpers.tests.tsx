import { getErrorMessage } from "./helpers";

jest.mock("axios", () => {
  const actual = jest.requireActual("axios");
  return {
    ...actual,
    isAxiosError: () => true,
  };
});

const errWithReason = (reason: string) => ({
  response: { status: 409, data: { errors: [{ name: "Error", reason }] } },
});

describe("getErrorMessage", () => {
  it("passes through the Firefox/ESR conflict message without doubling the prefix", () => {
    const err = errWithReason(
      "Couldn't add software. Only one of Mozilla Firefox or Mozilla Firefox ESR can be added to the same fleet."
    );

    expect(getErrorMessage(err)).toBe(
      "Couldn't add software. Only one of Mozilla Firefox or Mozilla Firefox ESR can be added to the same fleet."
    );
  });

  it("prefixes a generic reason", () => {
    const err = errWithReason("Something went wrong");

    expect(getErrorMessage(err)).toBe("Couldn't add. Something went wrong.");
  });
});
