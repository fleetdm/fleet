import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockAndroidCert } from "__mocks__/certificatesMock";

import ViewCertificateModal from "./ViewCertificateModal";

const mockOnExit = jest.fn();

const renderModal = (cert = createMockAndroidCert()) => {
  const render = createCustomRenderer();
  return render(<ViewCertificateModal cert={cert} onExit={mockOnExit} />);
};

describe("ViewCertificateModal", () => {
  beforeEach(() => {
    mockOnExit.mockClear();
  });

  it("renders the certificate name, CA, added time, and subject name", () => {
    const cert = createMockAndroidCert({
      name: "Zero trust certificate",
      certificate_authority_name: "PRODUCTION_SCEP_SERVER",
      subject_name: "CN=test@example.com, O=Example Inc.",
    });
    renderModal(cert);

    expect(screen.getByText("Zero trust certificate")).toBeInTheDocument();
    expect(screen.getByText("Certificate authority")).toBeInTheDocument();
    expect(screen.getByText("PRODUCTION_SCEP_SERVER")).toBeInTheDocument();
    expect(screen.getByText("Added")).toBeInTheDocument();
    expect(screen.getByText("Subject name (SN)")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("CN=test@example.com, O=Example Inc.")
    ).toBeInTheDocument();
  });

  it("hides the SAN section when the certificate has no subject alternative name", () => {
    renderModal(createMockAndroidCert({ subject_alternative_name: undefined }));
    expect(
      screen.queryByText("Subject alternative name (SAN)")
    ).not.toBeInTheDocument();
  });

  it("renders the SAN section when present", () => {
    renderModal(
      createMockAndroidCert({
        subject_alternative_name:
          "UPN=test@example.com, EMAIL=test@example.com",
      })
    );
    expect(
      screen.getByText("Subject alternative name (SAN)")
    ).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("UPN=test@example.com, EMAIL=test@example.com")
    ).toBeInTheDocument();
  });

  it("calls onExit when Done is clicked", async () => {
    const { user } = renderModal();
    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(mockOnExit).toHaveBeenCalledTimes(1);
  });
});
