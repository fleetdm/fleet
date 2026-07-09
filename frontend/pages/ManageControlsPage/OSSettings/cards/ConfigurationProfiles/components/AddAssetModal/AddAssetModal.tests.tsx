import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import mdmAPI from "services/entities/mdm";
import { notify } from "components/ToastNotification";

import AddAssetModal from "./AddAssetModal";

jest.mock("services/entities/mdm", () => ({
  __esModule: true,
  default: {
    uploadAsset: jest.fn(),
  },
}));

const render = createCustomRenderer();

describe("AddAssetModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the upload copy and disables Add asset until a file is chosen", () => {
    render(
      <AddAssetModal
        currentTeamId={0}
        onUpload={noop}
        setShowModal={noop as any}
      />
    );

    expect(
      screen.getByText(
        /only asset declarations \(com\.apple\.asset\) are supported/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Upload asset")).toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Add asset" })).toBeDisabled();
  });

  it("uploads the selected file and calls onUpload", async () => {
    (mdmAPI.uploadAsset as jest.Mock).mockResolvedValue({
      asset_uuid: "abc-123",
    });
    const onUpload = jest.fn();

    const { user, container } = render(
      <AddAssetModal
        currentTeamId={2}
        onUpload={onUpload}
        setShowModal={noop as any}
      />
    );

    const file = new File(['{"Type":"com.apple.asset.data"}'], "asset.json", {
      type: "application/json",
    });
    const input = container.querySelector("#upload-asset") as HTMLInputElement;
    await user.upload(input, file);

    const addButton = screen.getByRole("button", { name: "Add asset" });
    expect(addButton).toBeEnabled();

    await user.click(addButton);

    await waitFor(() => {
      expect(mdmAPI.uploadAsset).toHaveBeenCalledWith({
        file,
        teamId: 2,
      });
    });
    expect(onUpload).toHaveBeenCalled();
  });

  it("surfaces the API error reason when the upload fails", async () => {
    const reason =
      'An asset with the identifier "EB13EE2B" already exists for this team';
    (mdmAPI.uploadAsset as jest.Mock).mockRejectedValue({
      response: { data: { errors: [{ name: "base", reason }] } },
    });
    const errorSpy = jest.spyOn(notify, "error");

    const { user, container } = render(
      <AddAssetModal
        currentTeamId={2}
        onUpload={jest.fn()}
        setShowModal={noop as any}
      />
    );

    const file = new File(["{}"], "asset.json", { type: "application/json" });
    const input = container.querySelector("#upload-asset") as HTMLInputElement;
    await user.upload(input, file);
    await user.click(screen.getByRole("button", { name: "Add asset" }));

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith(reason, expect.anything());
    });

    errorSpy.mockRestore();
  });
});
