import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import ManagedAccountCheckbox from "./ManagedAccountCheckbox";

describe("ManagedAccountCheckbox", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  it("reports changes to the caller", async () => {
    const onChange = jest.fn();
    const { user } = render(
      <ManagedAccountCheckbox
        value={false}
        onChange={onChange}
        disabled={false}
      />
    );

    await user.click(
      screen.getByRole("checkbox", { name: "Create hidden admin" })
    );

    expect(onChange).toHaveBeenCalledWith(true);
  });

  it("does not report changes while disabled", async () => {
    const onChange = jest.fn();
    const { user } = render(
      <ManagedAccountCheckbox value={false} onChange={onChange} disabled />
    );

    await user.click(
      screen.getByRole("checkbox", { name: "Create hidden admin" })
    );

    expect(onChange).not.toHaveBeenCalled();
  });
});
