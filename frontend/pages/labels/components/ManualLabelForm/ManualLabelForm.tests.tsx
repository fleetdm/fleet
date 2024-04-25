import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockHost from "__mocks__/hostMock";

import ManualLabelForm, {
  LABEL_TARGET_HOSTS_INPUT_LABEL,
} from "./ManualLabelForm";

describe("ManualLabelForm", () => {
  it("should render a Select Hosts input", () => {
    const render = createCustomRenderer({ withBackendMock: true });

    render(<ManualLabelForm onSave={noop} onCancel={noop} />);

    expect(
      screen.getByText(LABEL_TARGET_HOSTS_INPUT_LABEL)
    ).toBeInTheDocument();
  });

  it("should pass up the form data when the form is submitted and valid", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const onSave = jest.fn();

    const name = "Test Name";
    const description = "Test Description";
    const targetedHosts = [createMockHost()];

    const { user } = render(
      <ManualLabelForm
        onSave={onSave}
        onCancel={noop}
        defaultTargetedHosts={targetedHosts}
      />
    );

    await user.type(screen.getByLabelText("Name"), name);
    await user.type(screen.getByLabelText("Description"), description);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith({
      name,
      description,
      targetedHosts,
    });
  });
});
