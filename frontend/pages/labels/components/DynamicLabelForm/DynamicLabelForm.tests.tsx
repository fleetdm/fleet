import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import DynamicLabelForm from "./DynamicLabelForm";

describe("DynamicLabelForm", () => {
  it("should render the Fleet Ace and Select Platform input", () => {
    render(<DynamicLabelForm onSave={noop} onCancel={noop} />);

    expect(screen.getByText("Query")).toBeInTheDocument();
    expect(screen.getByText("All platforms")).toBeInTheDocument();
  });

  it("should pass up the form data when the form is submitted and valid", async () => {
    const onSave = jest.fn();

    const name = "Test Name";
    const description = "Test Description";
    const query = "SELECT * FROM users;";
    const platform = "darwin";

    const { user } = renderWithSetup(
      <DynamicLabelForm
        onSave={onSave}
        onCancel={noop}
        defaultQuery={query}
        defaultPlatform={platform}
      />
    );

    await user.type(screen.getByLabelText("Name"), name);
    await user.type(screen.getByLabelText("Description"), description);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith({
      name,
      description,
      query,
      platform,
    });
  });
});
