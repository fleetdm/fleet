import { getInstallErrorMessage } from "./helpers";

const makeErr = (reason: string) => ({
  response: {
    data: {
      errors: [{ name: "base", reason }],
    },
  },
});

describe("getInstallErrorMessage", () => {
  it("returns fleetd-specific message", () => {
    const result = getInstallErrorMessage(
      makeErr("host has fleetd installed from another source")
    );
    expect(result).toContain("fleetd installed");
  });

  it("returns macOS-only install message", () => {
    const result = getInstallErrorMessage(
      makeErr("software can be installed only on darwin")
    );
    expect(result).toBe(
      "Couldn't install. Software can be installed only on macOS."
    );
  });

  it("returns MDM turned off message as-is", () => {
    const result = getInstallErrorMessage(makeErr("MDM is turned off."));
    expect(result).toBe("MDM is turned off");
  });

  it("returns no available licenses message as-is", () => {
    const result = getInstallErrorMessage(makeErr("No available licenses."));
    expect(result).toBe("No available licenses");
  });

  it("returns unresolvable Fleet variable message", () => {
    const result = getInstallErrorMessage(
      makeErr(
        "apple_mdm: unresolvable Fleet variable in managed app configuration"
      )
    );
    expect(result).toBe(
      "Couldn't install. Couldn't resolve a Fleet variable in the managed app configuration for this host."
    );
  });

  it("returns default message for unknown errors", () => {
    const result = getInstallErrorMessage(makeErr("something unexpected"));
    expect(result).toBe("Couldn't install. Please try again.");
  });

  it("returns default message when no reason can be extracted", () => {
    expect(getInstallErrorMessage({})).toBe(
      "Couldn't install. Please try again."
    );
  });
});
