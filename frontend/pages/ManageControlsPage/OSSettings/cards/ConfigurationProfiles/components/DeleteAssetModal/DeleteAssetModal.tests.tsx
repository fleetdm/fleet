import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";

import DeleteAssetModal from "./DeleteAssetModal";

const render = createCustomRenderer();

describe("DeleteAssetModal", () => {
  it("renders the referenced-asset warning copy", () => {
    render(
      <DeleteAssetModal
        assetUuid="abc-123"
        onCancel={noop}
        onDelete={noop}
        isDeleting={false}
      />
    );

    expect(
      screen.getByText(
        /Assets that are linked in a configuration profile will not be deleted/i
      )
    ).toBeInTheDocument();
  });

  it("calls onDelete with the asset uuid and onCancel", async () => {
    const onDelete = jest.fn();
    const onCancel = jest.fn();

    const { user } = render(
      <DeleteAssetModal
        assetUuid="abc-123"
        onCancel={onCancel}
        onDelete={onDelete}
        isDeleting={false}
      />
    );

    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(onDelete).toHaveBeenCalledWith("abc-123");

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalled();
  });
});
