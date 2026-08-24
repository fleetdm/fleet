import React from "react";
import { screen } from "@testing-library/react";

import { createMockVppApp } from "__mocks__/appleMdm";
import { createCustomRenderer } from "test/test-utils";

import SoftwareVppForm from "./SoftwareVppForm";

describe("SoftwareVppForm", () => {
  it("shows Self service after selecting an app to add", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(
      <SoftwareVppForm
        labels={[]}
        vppApps={[createMockVppApp()]}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
        onClickPreviewEndUserExperience={jest.fn()}
        teamId={1}
      />
    );

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Test App/ }));

    expect(screen.getByText("Self service")).toBeInTheDocument();
  });
});
