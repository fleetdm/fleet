import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { noop } from "lodash";

import { HostPlatform } from "interfaces/platform";
import { IGetHostCertificatesResponse } from "services/entities/hosts";
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

const renderTable = ({
  data = createMockGetHostCertificatesResponse(),
  hostPlatform = "darwin",
  showHelpText = false,
}: {
  data?: IGetHostCertificatesResponse;
  hostPlatform?: HostPlatform;
  showHelpText?: boolean;
} = {}) =>
  createCustomRenderer()(
    <CertificatesTable
      {...baseProps}
      data={data}
      hostPlatform={hostPlatform}
      showHelpText={showHelpText}
    />
  );

describe("CertificatesTable", () => {
  it("renders the platform-agnostic 'Scope' column header (replacing 'Keychain')", () => {
    renderTable();

    expect(screen.getByText("Scope")).toBeInTheDocument();
    expect(screen.queryByText("Keychain")).not.toBeInTheDocument();
  });

  it("shows macOS keychain help text on a darwin host", () => {
    renderTable({ hostPlatform: "darwin", showHelpText: true });

    expect(screen.getByText(/login \(user\) keychain/i)).toBeInTheDocument();
  });

  it("shows Personal certificate store help text on a windows host", () => {
    renderTable({ hostPlatform: "windows", showHelpText: true });

    expect(screen.getByText(/Personal certificate store/i)).toBeInTheDocument();
  });

  it("renders a user-scoped certificate as 'User' with its owning username", async () => {
    const { user } = renderTable({
      data: createMockGetHostCertificatesResponse({
        certificates: [
          createMockHostCertificate({ source: "user", username: "alice" }),
        ],
        count: 1,
      }),
      hostPlatform: "windows",
    });

    expect(screen.getByText("User")).toBeInTheDocument();
    // the owning username is surfaced in the scope cell's tooltip on hover
    await user.hover(screen.getByText("User"));
    expect(await screen.findByText("alice")).toBeInTheDocument();
  });

  it("renders a certificate present in two scopes as two distinct rows (shared id must not collapse)", () => {
    // Same certificate (same id) installed in both the System store and a user's
    // store comes back as two rows sharing host_certificates.id. They must each
    // render rather than collapsing on the shared id.
    renderTable({
      data: createMockGetHostCertificatesResponse({
        certificates: [
          createMockHostCertificate({
            id: 1,
            common_name: "shared.example.com",
            source: "system",
            username: "",
          }),
          createMockHostCertificate({
            id: 1,
            common_name: "shared.example.com",
            source: "user",
            username: "alice",
          }),
        ],
        count: 2,
      }),
      hostPlatform: "windows",
    });

    // Both scope cells render — without a per-scope row id the two same-id rows
    // collapse and only one scope would be shown.
    expect(screen.getByText("System")).toBeInTheDocument();
    expect(screen.getByText("User")).toBeInTheDocument();
  });
});
