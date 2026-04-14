import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";

import generateErrorTooltip from "./errorTooltipHelpers";

// Helper to render the JSX returned by generateErrorTooltip
const renderTooltip = (tooltip: React.ReactNode) => {
  return render(<div>{tooltip}</div>);
};

describe("generateErrorTooltip", () => {
  it("returns null for non-failed profiles", () => {
    const result = generateErrorTooltip(
      createMockHostMdmProfile({ status: "verified" })
    );
    expect(result).toBeNull();
  });

  it("returns null for failed profiles with no detail", () => {
    const result = generateErrorTooltip(
      createMockHostMdmProfile({ status: "failed", detail: "" })
    );
    expect(result).toBeNull();
  });

  it("formats a windows profile error with key-value pairs", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        platform: "windows",
        status: "failed",
        detail:
          "starting encryption: encrypt(C:): error code returned during encryption: -2147024809, error 2: This is another error",
      })
    );

    renderTooltip(tooltip);

    const firstErrorKey = screen.getByText(
      (content) => content === "starting encryption:"
    );
    const firstErrorValue = screen.getByText(
      (content) =>
        content ===
        "encrypt(C:): error code returned during encryption: -2147024809,"
    );

    expect(firstErrorKey).toBeInTheDocument();
    expect(firstErrorKey.tagName.toLowerCase()).toBe("b");
    expect(firstErrorValue).toBeInTheDocument();

    const secondErrorKey = screen.getByText(
      (content) => content === "error 2:"
    );
    const secondErrorValue = screen.getByText(
      (content) => content === "This is another error"
    );

    expect(secondErrorKey).toBeInTheDocument();
    expect(secondErrorKey.tagName.toLowerCase()).toBe("b");
    expect(secondErrorValue).toBeInTheDocument();
  });

  it("renders a tooltip link for IDP email errors", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        status: "failed",
        detail: "There is no IdP email for this host.",
      })
    );

    renderTooltip(tooltip);

    expect(screen.getByText(/Learn more/)).toBeInTheDocument();
    expect(screen.getByText(/Learn more/).tagName.toLowerCase()).toBe("a");
  });

  it("formats a custom SCEP certificate error", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Fleet couldn't populate $FLEET_VAR_CUSTOM_SCEP_URL_SCEP_WIFI because SCEP_WIFI certificate authority doesn't exist.`,
      })
    );

    renderTooltip(tooltip);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/add it and resend the configuration profile/)
    ).toBeInTheDocument();
  });

  it("formats a DigiCert profile ID error (410)", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert for WIFI_CERTIFICATE. unexpected DigiCert status code for POST request: 410, errors: Profile with id {test-id} was deleted`,
      })
    );

    renderTooltip(tooltip);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("WIFI_CERTIFICATE")).toBeInTheDocument();
    expect(screen.getByText("Profile GUID")).toBeInTheDocument();
  });

  it("formats a DigiCert deleted/suspended profile error (400)", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert for WIFI_CERTIFICATE. unexpected DigiCert status code for POST request: 400, errors: Enrollment creation and Certificate issuance/renewal for deleted or suspended Profile are not supported.
          Please contact system Administrator.`,
      })
    );

    renderTooltip(tooltip);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("WIFI_CERTIFICATE")).toBeInTheDocument();
    expect(screen.getByText("Profile GUID")).toBeInTheDocument();
  });

  it("formats a DigiCert token error", () => {
    const tooltip = generateErrorTooltip(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert. The API token configured in DIGICERT_TEST certificate authority is invalid.`,
      })
    );

    renderTooltip(tooltip);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("DIGICERT_TEST")).toBeInTheDocument();
    expect(screen.getByText("API token")).toBeInTheDocument();
  });

  it("returns the raw detail string for unrecognized darwin errors", () => {
    const result = generateErrorTooltip(
      createMockHostMdmProfile({
        platform: "darwin",
        status: "failed",
        detail: "Some unknown error occurred",
      })
    );

    expect(result).toBe("Some unknown error occurred");
  });
});
