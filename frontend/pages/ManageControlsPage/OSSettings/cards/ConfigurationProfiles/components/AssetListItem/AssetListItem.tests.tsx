import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import { IMdmAsset } from "interfaces/mdm";

import AssetListItem from "./AssetListItem";

const render = createCustomRenderer();

const asset: IMdmAsset = {
  asset_uuid: "u1",
  name: "JSON Asset",
  identifier: "com.example.asset1",
  created_at: "2024-01-01T00:00:00Z",
  uploaded_at: "2024-01-01T00:00:00Z",
  checksum: "abc",
};

describe("AssetListItem", () => {
  it("renders the identifier, a copy button, and download/delete actions", () => {
    render(
      <AssetListItem asset={asset} onClickDelete={noop} isTechnician={false} />
    );

    expect(screen.getByText("com.example.asset1")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Copy com.example.asset1" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Download JSON Asset" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Delete JSON Asset" })
    ).toBeInTheDocument();
  });

  it("calls onClickDelete with the asset when the delete button is clicked", async () => {
    const onClickDelete = jest.fn();
    const { user } = render(
      <AssetListItem
        asset={asset}
        onClickDelete={onClickDelete}
        isTechnician={false}
      />
    );

    await user.click(screen.getByRole("button", { name: "Delete JSON Asset" }));
    expect(onClickDelete).toHaveBeenCalledWith(asset);
  });

  it("hides the delete action for technicians", () => {
    render(<AssetListItem asset={asset} onClickDelete={noop} isTechnician />);

    expect(
      screen.queryByRole("button", { name: "Delete JSON Asset" })
    ).not.toBeInTheDocument();
    // download remains available
    expect(
      screen.getByRole("button", { name: "Download JSON Asset" })
    ).toBeInTheDocument();
  });

  it("disables the delete action in GitOps mode", () => {
    const renderGitOps = createCustomRenderer({
      context: {
        app: {
          config: { gitops: { gitops_mode_enabled: true } } as any,
        },
      },
    });

    renderGitOps(
      <AssetListItem asset={asset} onClickDelete={noop} isTechnician={false} />
    );

    expect(
      screen.getByRole("button", { name: "Delete JSON Asset" })
    ).toBeDisabled();
  });
});
