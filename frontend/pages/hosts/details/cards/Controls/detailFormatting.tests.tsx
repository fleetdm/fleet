import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";
import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";

import {
  getDetailGuidance,
  getDetailText,
  IDetailGuidance,
} from "./detailFormatting";

const renderGuidance = (guidance: IDetailGuidance | null) => {
  return render(<div>{guidance?.message}</div>);
};

describe("getDetailGuidance", () => {
  it("returns null for non-failed profiles", () => {
    const result = getDetailGuidance(
      createMockHostMdmProfile({ status: "verified" })
    );
    expect(result).toBeNull();
  });

  it("returns null for failed profiles with no detail", () => {
    const result = getDetailGuidance(
      createMockHostMdmProfile({ status: "failed", detail: "" })
    );
    expect(result).toBeNull();
  });

  it("returns null for an unrecognized error, leaving the raw detail to stand alone", () => {
    const result = getDetailGuidance(
      createMockHostMdmProfile({
        platform: "darwin",
        status: "failed",
        detail: "Some unknown error occurred",
      })
    );

    expect(result).toBeNull();
  });

  it("renders a learn more link for IdP email errors", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        status: "failed",
        detail: "There is no IdP email for this host.",
      })
    );

    renderGuidance(guidance);

    expect(screen.getByText(/Learn more/)).toBeInTheDocument();
    expect(screen.getByText(/Learn more/).tagName.toLowerCase()).toBe("a");
    // The message quotes the detail, so the copyable block would repeat it.
    expect(guidance?.supersedesDetail).toBe(true);
  });

  it("renders a learn more link for Android profile errors", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        platform: "android",
        status: "failed",
        detail: "Some settings couldn't apply to a host.",
      })
    );

    renderGuidance(guidance);

    expect(screen.getByText(/Learn more/)).toBeInTheDocument();
    expect(guidance?.supersedesDetail).toBe(true);
  });

  it("formats a custom SCEP certificate error", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Fleet couldn't populate $FLEET_VAR_CUSTOM_SCEP_URL_SCEP_WIFI because SCEP_WIFI certificate authority doesn't exist.`,
      })
    );

    renderGuidance(guidance);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/add it and resend the configuration profile/)
    ).toBeInTheDocument();
    // The rewrite drops the underlying error text, so the block still earns
    // its place.
    expect(guidance?.supersedesDetail).toBe(false);
  });

  it("formats a DigiCert profile ID error (410)", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert for WIFI_CERTIFICATE. unexpected DigiCert status code for POST request: 410, errors: Profile with id {test-id} was deleted`,
      })
    );

    renderGuidance(guidance);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("WIFI_CERTIFICATE")).toBeInTheDocument();
    expect(screen.getByText("Profile GUID")).toBeInTheDocument();
  });

  it("formats a DigiCert deleted/suspended profile error (400)", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert for WIFI_CERTIFICATE. unexpected DigiCert status code for POST request: 400, errors: Enrollment creation and Certificate issuance/renewal for deleted or suspended Profile are not supported.
          Please contact system Administrator.`,
      })
    );

    renderGuidance(guidance);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("WIFI_CERTIFICATE")).toBeInTheDocument();
    expect(screen.getByText("Profile GUID")).toBeInTheDocument();
  });

  it("formats a DigiCert token error", () => {
    const guidance = getDetailGuidance(
      createMockHostMdmProfile({
        status: "failed",
        detail: `Couldn't get certificate from DigiCert. The API token configured in DIGICERT_TEST certificate authority is invalid.`,
      })
    );

    renderGuidance(guidance);

    expect(
      screen.getByText("Settings > Integrations > Certificates")
    ).toBeInTheDocument();
    expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
    expect(screen.getByText("DIGICERT_TEST")).toBeInTheDocument();
    expect(screen.getByText("API token")).toBeInTheDocument();
  });
});

// Every reason the Fleet Android app reports gets restated without SCEP, CSR, or key pair
// jargon, since these messages also reach the end user's My device page.
describe("getDetailGuidance for android certificates", () => {
  const failedCert = (detail: string) =>
    createMockHostMdmProfile({
      profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
      name: "BeyondCorp",
      platform: "android",
      operation_type: "install",
      status: "failed",
      certificate_template_id: 4,
      detail,
    });

  it.each([
    [
      "Network error during SCEP enrollment: Failed to communicate with SCEP server",
      "Fleet couldn't reach the certificate authority.",
    ],
    [
      "SCEP enrollment failed: challenge rejected",
      "The certificate authority rejected the request.",
    ],
    [
      "Certificate validation failed: bad chain",
      "The host couldn't validate the certificate from the certificate authority.",
    ],
    [
      "Failed to generate key pair: keystore error",
      "The host couldn't generate a private key.",
    ],
    [
      "Failed to create CSR: bad subject",
      "The host couldn't create a certificate signing request.",
    ],
    [
      "Invalid configuration: missing subject name",
      "The certificate configuration isn't valid.",
    ],
    [
      "Certificate installation failed for alias 'BeyondCorp': installKeyPair returned false",
      "The host couldn't install the certificate.",
    ],
  ])("restates %j in plain language", (detail, message) => {
    const guidance = getDetailGuidance(failedCert(detail));

    renderGuidance(guidance);

    expect(screen.getByText(message)).toBeInTheDocument();
    // The rewrite drops the app's diagnostic text, so the raw detail stays below it.
    expect(guidance?.supersedesDetail).toBe(false);
  });

  it("returns null for an unrecognized failure, leaving the reported text to stand alone", () => {
    expect(
      getDetailGuidance(failedCert("Unexpected error during enrollment: null"))
    ).toBeNull();
  });
});

describe("getDetailText", () => {
  it("returns an empty string when there is no detail", () => {
    expect(getDetailText(createMockHostMdmProfile({ detail: "" }))).toBe("");
  });

  it("returns the raw detail for non-Windows profiles", () => {
    expect(
      getDetailText(
        createMockHostMdmProfile({
          platform: "darwin",
          detail: "Some unknown error occurred",
        })
      )
    ).toBe("Some unknown error occurred");
  });

  it("breaks a Windows key-value detail into one result per line", () => {
    expect(
      getDetailText(
        createMockHostMdmProfile({
          platform: "windows",
          detail:
            "./Device/Vendor/MSFT/Policy/Config/Fleet/A: status 200, ./Device/Vendor/MSFT/Policy/Config/Fleet/B: status 500",
        })
      )
    ).toBe(
      "./Device/Vendor/MSFT/Policy/Config/Fleet/A: status 200\n./Device/Vendor/MSFT/Policy/Config/Fleet/B: status 500"
    );
  });

  it("leaves a Windows certificate install error unsplit", () => {
    const detail = `Couldn't install certificate. The "WINSCEPTEST" certificate authority challenge includes characters Windows doesn't support. Allowed: letters, numbers, spaces, and ' ( ) + , - . / : = ?`;

    expect(
      getDetailText(
        createMockHostMdmProfile({
          platform: "windows",
          status: "failed",
          detail,
        })
      )
    ).toBe(detail);
  });

  it("leaves a BitLocker error unsplit", () => {
    const detail =
      "BitLocker: starting encryption: encrypt(C:): error code returned during encryption: -2147024809. Fleet will retry automatically.";

    expect(
      getDetailText(
        createMockHostMdmProfile({
          platform: "windows",
          status: "failed",
          detail,
        })
      )
    ).toBe(detail);
  });
});
