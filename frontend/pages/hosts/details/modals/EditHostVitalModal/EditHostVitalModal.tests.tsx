import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import customHostVitalsAPI from "services/entities/custom_host_vitals";

import EditHostVitalModal from "./EditHostVitalModal";

jest.mock("services/entities/custom_host_vitals");

const vital = {
  custom_host_vital_id: 5,
  name: "Asset tag",
  value: "FLEET-001234",
};

describe("EditHostVitalModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("renders the static title with the vital name as the field label, prefilled with its value", () => {
    render(
      <EditHostVitalModal
        hostId={7}
        vital={vital}
        onCancel={jest.fn()}
        onSave={jest.fn()}
      />
    );

    expect(screen.getByText("Edit host vital")).toBeVisible();
    expect(screen.getByRole("textbox", { name: "Asset tag" })).toHaveValue(
      "FLEET-001234"
    );
  });

  it("saves the edited value and calls onSave on success", async () => {
    (customHostVitalsAPI.updateHostCustomHostVitalValue as jest.Mock).mockResolvedValue(
      undefined
    );
    const onSave = jest.fn();

    const { user } = render(
      <EditHostVitalModal
        hostId={7}
        vital={vital}
        onCancel={jest.fn()}
        onSave={onSave}
      />
    );

    const input = screen.getByRole("textbox", { name: "Asset tag" });
    await user.clear(input);
    await user.type(input, "FLEET-999");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(
        customHostVitalsAPI.updateHostCustomHostVitalValue
      ).toHaveBeenCalledWith(7, 5, "FLEET-999");
    });
    await waitFor(() => {
      expect(onSave).toHaveBeenCalled();
    });
  });

  it("does not call onSave when the update errors", async () => {
    (customHostVitalsAPI.updateHostCustomHostVitalValue as jest.Mock).mockRejectedValue(
      new Error("boom")
    );
    const onSave = jest.fn();

    const { user } = render(
      <EditHostVitalModal
        hostId={7}
        vital={vital}
        onCancel={jest.fn()}
        onSave={onSave}
      />
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(
        customHostVitalsAPI.updateHostCustomHostVitalValue
      ).toHaveBeenCalledWith(7, 5, "FLEET-001234");
    });
    expect(onSave).not.toHaveBeenCalled();
  });

  it("calls onCancel when the Cancel button is clicked", async () => {
    const onCancel = jest.fn();

    const { user } = render(
      <EditHostVitalModal
        hostId={7}
        vital={vital}
        onCancel={onCancel}
        onSave={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onCancel).toHaveBeenCalled();
  });
});
