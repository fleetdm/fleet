import { isAddHostsAvailable } from "./helpers";

describe("isAddHostsAvailable", () => {
  it("is available when one-time enroll secrets are off, with or without a secret", () => {
    expect(
      isAddHostsAvailable({
        useOneTimeEnrollSecrets: false,
        isLoadingSecrets: false,
        hasEnrollSecret: false,
      })
    ).toBe(true);
    expect(
      isAddHostsAvailable({
        useOneTimeEnrollSecrets: false,
        isLoadingSecrets: false,
        hasEnrollSecret: true,
      })
    ).toBe(true);
  });

  it("is available when one-time enroll secrets are on and a shared secret exists", () => {
    expect(
      isAddHostsAvailable({
        useOneTimeEnrollSecrets: true,
        isLoadingSecrets: false,
        hasEnrollSecret: true,
      })
    ).toBe(true);
  });

  it("stays available while secrets are still loading", () => {
    expect(
      isAddHostsAvailable({
        useOneTimeEnrollSecrets: true,
        isLoadingSecrets: true,
        hasEnrollSecret: false,
      })
    ).toBe(true);
  });

  it("is unavailable when one-time enroll secrets are on and no shared secret exists", () => {
    expect(
      isAddHostsAvailable({
        useOneTimeEnrollSecrets: true,
        isLoadingSecrets: false,
        hasEnrollSecret: false,
      })
    ).toBe(false);
  });
});
