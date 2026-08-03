import { REQUEST_TIMEOUT_ERROR_MESSAGE } from "../../helpers";
import { getErrorMessage } from "./helpers";

jest.mock("axios", () => {
  const actual = jest.requireActual("axios");
  return {
    ...actual,
    isAxiosError: () => true,
  };
});

const errWithStatus = (status: number, reason: string) => ({
  response: { status, data: { errors: [{ name: "base", reason }] } },
});

const responseWithStatus = (status: number, reason: string) => ({
  status,
  data: { errors: [{ name: "base", reason }] },
});

const errWithReason = (reason: string) => errWithStatus(409, reason);

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

  it("shows the friendly timeout message on 504", () => {
    expect(getErrorMessage(errWithStatus(504, "anything"))).toBe(
      REQUEST_TIMEOUT_ERROR_MESSAGE
    );
  });

  it("shows the friendly timeout message when the service rejects with a 504 response", () => {
    expect(getErrorMessage(responseWithStatus(504, "anything"))).toBe(
      REQUEST_TIMEOUT_ERROR_MESSAGE
    );
  });

  it("shows the friendly timeout message on 499 (upstream cancel) and never leaks 'context canceled'", () => {
    const msg = getErrorMessage(
      errWithStatus(
        499,
        'downloading app installer: reading installer "x" contents: context canceled'
      )
    );
    expect(msg).toBe(REQUEST_TIMEOUT_ERROR_MESSAGE);
    expect(msg).not.toMatch(/context canceled/i);
  });

  it("still shows the friendly timeout message on 408", () => {
    expect(getErrorMessage(errWithStatus(408, "x"))).toBe(
      REQUEST_TIMEOUT_ERROR_MESSAGE
    );
  });
});
