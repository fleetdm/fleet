import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";

import GeneratedProfileSummary from "./GeneratedProfileSummary";

const renderSummary = (
  props: Partial<React.ComponentProps<typeof GeneratedProfileSummary>> = {}
) => {
  const onChangeName = jest.fn();
  const onChangeDescription = jest.fn();
  render(
    <GeneratedProfileSummary
      name="Disable AirDrop"
      description="Sets `allowAirDrop` to `false`."
      onChangeName={onChangeName}
      onChangeDescription={onChangeDescription}
      {...props}
    />
  );
  return { onChangeName, onChangeDescription };
};

describe("GeneratedProfileSummary", () => {
  it("shows the name as a heading and inline code in the description as chips", () => {
    renderSummary();
    expect(
      screen.getByRole("heading", { name: "Disable AirDrop" })
    ).toBeInTheDocument();
    expect(screen.getByText("allowAirDrop").tagName).toBe("CODE");
    expect(screen.getByText("false").tagName).toBe("CODE");
    expect(screen.queryByText(/`/)).not.toBeInTheDocument();
  });

  it("edits the name in place and saves it on Enter", () => {
    const { onChangeName } = renderSummary();
    fireEvent.click(screen.getByRole("button", { name: "Edit name" }));
    const field = screen.getByRole("textbox", { name: "Name" });
    fireEvent.change(field, { target: { value: "  Block AirDrop  " } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChangeName).toHaveBeenCalledWith("Block AirDrop");
  });

  it("drops an edit on Escape", () => {
    const { onChangeDescription } = renderSummary();
    fireEvent.click(screen.getByRole("button", { name: "Edit description" }));
    const field = screen.getByRole("textbox", { name: "Description" });
    fireEvent.change(field, { target: { value: "Something else" } });
    fireEvent.keyDown(field, { key: "Escape" });
    expect(onChangeDescription).not.toHaveBeenCalled();
    expect(screen.getByText("allowAirDrop")).toBeInTheDocument();
  });

  it("saves the description when the field loses focus", () => {
    const { onChangeDescription } = renderSummary();
    fireEvent.click(screen.getByRole("button", { name: "Edit description" }));
    const field = screen.getByRole("textbox", { name: "Description" });
    fireEvent.change(field, { target: { value: "Turns off AirDrop." } });
    fireEvent.blur(field);
    expect(onChangeDescription).toHaveBeenCalledWith("Turns off AirDrop.");
  });

  it("keeps 1Password from offering to fill the name and description", () => {
    renderSummary();
    fireEvent.click(screen.getByRole("button", { name: "Edit name" }));
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveAttribute(
      "data-1p-ignore"
    );
    fireEvent.click(screen.getByRole("button", { name: "Edit description" }));
    expect(
      screen.getByRole("textbox", { name: "Description" })
    ).toHaveAttribute("data-1p-ignore");
  });

  it("shows empty values as fields to fill in", () => {
    renderSummary({ name: "", description: "" });
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveValue("");
    expect(screen.getByRole("textbox", { name: "Description" })).toHaveValue(
      ""
    );
  });

  it("shows placeholders while the values are pending", () => {
    renderSummary({ isNamePending: true, isDescriptionPending: true });
    expect(screen.getByTestId("name-placeholder")).toBeInTheDocument();
    expect(screen.getByTestId("description-placeholder")).toBeInTheDocument();
    expect(screen.queryByRole("heading")).not.toBeInTheDocument();
  });
});
