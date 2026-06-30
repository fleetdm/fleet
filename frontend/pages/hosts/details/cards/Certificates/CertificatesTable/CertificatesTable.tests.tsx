import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { noop } from "lodash";

import {
  createMockGetHostCertificatesResponse,
  createMockHostCertificate,
} from "__mocks__/certificatesMock";

import CertificatesTable from "./CertificatesTable";

const baseProps = {
  page: 0,
  pageSize: 20,
  sortHeader: "common_name",
  sortDirection: "asc",
  onSelectCertificate: noop,
  onNextPage: noop,
  onPreviousPage: noop,
  onSortChange: noop,
};

describe("CertificatesTable", () => {
  it("renders the platform-agnostic 'Scope' column header (replacing 'Keychain')", () => {
    const render = createCustomRenderer();
    render(
      <CertificatesTable
        {...baseProps}
        data={createMockGetHostCertificatesResponse()}
        hostPlatform="darwin"
        showHelpText={false}
      />
    );

    expect(screen.getByText("Scope")).toBeInTheDocument();
    expect(screen.queryByText("Keychain")).not.toBeInTheDocument();
  });

  it("shows macOS keychain help text on a darwin host", () => {
    const render = createCustomRenderer();
    render(
      <CertificatesTable
        {...baseProps}
        data={createMockGetHostCertificatesResponse()}
        hostPlatform="darwin"
        showHelpText
      />
    );

    expect(screen.getByText(/login \(user\) keychain/i)).toBeInTheDocument();
  });

  it("shows Personal certificate store help text on a windows host", () => {
    const render = createCustomRenderer();
    render(
      <CertificatesTable
        {...baseProps}
        data={createMockGetHostCertificatesResponse()}
        hostPlatform="windows"
        showHelpText
      />
    );

    expect(screen.getByText(/Personal certificate store/i)).toBeInTheDocument();
  });

  it("renders the User scope for a user-scoped certificate", () => {
    const render = createCustomRenderer();
    render(
      <CertificatesTable
        {...baseProps}
        data={createMockGetHostCertificatesResponse({
          certificates: [
            createMockHostCertificate({ source: "user", username: "alice" }),
          ],
          count: 1,
        })}
        hostPlatform="windows"
        showHelpText={false}
      />
    );

    expect(screen.getByText("User")).toBeInTheDocument();
  });
});
