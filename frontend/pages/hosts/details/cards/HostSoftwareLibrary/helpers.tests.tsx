import { render } from "@testing-library/react";
import React from "react";

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

  it("surfaces the missing Managed Apple ID message as-is", () => {
    const result = getInstallErrorMessage(
      makeErr(
        "Couldn't install. Fleet hasn't received a Managed Apple ID for this host yet. Please wait a few minutes after enrollment and try again."
      )
    );
    expect(result).toBe(
      "Couldn't install. Fleet hasn't received a Managed Apple ID for this host yet. Please wait a few minutes after enrollment and try again"
    );
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

  it("surfaces the expired VPP token name", () => {
    const result = getInstallErrorMessage(
      makeErr(
        `Couldn't install. VPP token "Acme US" is expired. Please renew it and try again.`
      )
    );
    const { container } = render(<>{result}</>);
    expect(container.textContent).toContain(`VPP token "Acme US" is expired`);
    expect(container.textContent).toContain(
      "Settings > Integration > Volume Purchasing Program"
    );
    expect(container.textContent).not.toContain(
      "Please renew it and try again"
    );
  });

  it("surfaces an expired VPP token whose name contains quotes", () => {
    // Go's %q escapes embedded quotes as \", which arrives verbatim in the
    // JSON response reason. The UI shouldn't silently fall back to the
    // generic install error.
    const result = getInstallErrorMessage(
      makeErr(
        `Couldn't install. VPP token "Acme \\"US\\"" is expired. Please renew it and try again.`
      )
    );
    const { container } = render(<>{result}</>);
    expect(container.textContent).toContain("is expired");
    expect(container.textContent).toContain(
      "Settings > Integration > Volume Purchasing Program"
    );
    expect(container.textContent).not.toBe(
      "Couldn't install. Please try again."
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
