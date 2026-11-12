import { screen, within } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockVppToken } from "__mocks__/mdmMock";
import { IMdmVppToken } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";
import { internationalTimeFormat } from "utilities/helpers";

import VppTable from "./VppTable";

const createToken = (overrides: Partial<IMdmVppToken> = {}): IMdmVppToken =>
  createMockVppToken({
    org_name: "Zeta",
    renew_date: "2099-11-29T00:00:00Z",
    created_at: new Date().toISOString(),
    ...overrides,
  });

const renderTable = (vppTokens: IMdmVppToken[], gitOpsModeEnabled = false) => {
  const config = createMockConfig();
  config.gitops.gitops_mode_enabled = gitOpsModeEnabled;
  const render = createCustomRenderer({ context: { app: { config } } });
  return render(
    <VppTable
      vppTokens={vppTokens}
      onEditTokenTeam={jest.fn()}
      onRenewToken={jest.fn()}
      onDeleteToken={jest.fn()}
    />
  );
};

describe("VppTable", () => {
  it("renders rows in the order the API returned them", () => {
    renderTable([
      createToken({ id: 1, org_name: "Zeta" }),
      createToken({ id: 2, org_name: "Acme" }),
    ]);

    const rows = screen.getAllByRole("row").slice(1);
    expect(within(rows[0]).getByText("Zeta")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Acme")).toBeInTheDocument();
  });

  it("renders the country with its flag and name, and --- when empty", () => {
    renderTable([
      createToken({ id: 1, org_name: "Zeta", country_code: "jp" }),
      createToken({ id: 2, org_name: "Acme", country_code: "" }),
    ]);

    const rows = screen.getAllByRole("row").slice(1);
    expect(within(rows[0]).getByText("🇯🇵 Japan")).toBeInTheDocument();
    expect(within(rows[1]).queryByText("🇯🇵 Japan")).not.toBeInTheDocument();
    expect(within(rows[1]).getByText("---")).toBeInTheDocument();
  });

  it("renders Added to Fleet as a relative time, and --- when null", () => {
    renderTable([
      createToken({ id: 1, org_name: "Zeta" }),
      createToken({ id: 2, org_name: "Acme", created_at: null }),
    ]);

    expect(
      screen.getByRole("columnheader", { name: "Added to Fleet" })
    ).toBeInTheDocument();
    const rows = screen.getAllByRole("row").slice(1);
    expect(within(rows[0]).getByText(/second|minute|ago/i)).toBeInTheDocument();
    expect(within(rows[1]).getByText("---")).toBeInTheDocument();
  });

  it("shows the exact added time on hover", async () => {
    const createdAt = "2026-03-04T05:06:07Z";
    const { user } = renderTable([createToken({ created_at: createdAt })]);

    await user.hover(screen.getByText(/ago/i));

    expect(
      await screen.findByText(internationalTimeFormat(new Date(createdAt)))
    ).toBeInTheDocument();
  });

  it("disables only Edit fleets in GitOps mode", async () => {
    const { user } = renderTable([createToken()], true);

    await user.click(screen.getByText("Actions"));

    const isDisabled = (label: string) =>
      screen
        .getByText(label)
        .closest("[aria-disabled]")
        ?.getAttribute("aria-disabled") === "true";
    expect(isDisabled("Edit fleets")).toBe(true);
    expect(isDisabled("Renew")).toBe(false);
    expect(isDisabled("Delete")).toBe(false);
  });
});
