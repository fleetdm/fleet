import React from "react";
import { fireEvent } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { IFileDetails } from "utilities/file/fileUtils";

import FileDetails from "./FileDetails";

const defaultFileDetails = {
  name: "config.yaml",
  description: "File description",
} as IFileDetails;

const setup = (props = {}) =>
  renderWithSetup(
    <FileDetails
      graphicNames="file-pkg"
      fileDetails={defaultFileDetails}
      canEdit
      onFileSelect={jest.fn()}
      {...props}
    />
  );

describe("FileDetails", () => {
  it("renders file name and description and calls input.click when edit button is clicked (non-GitOps path)", async () => {
    const { user, container } = setup();

    expect(
      container.querySelector(".file-details__name")?.textContent
    ).toContain("config.yaml");
    expect(
      container.querySelector(".file-details__description")?.textContent
    ).toContain("File description");

    const fileInput = container.querySelector(
      "input[type='file']"
    ) as HTMLInputElement;
    const clickSpy = jest.spyOn(fileInput, "click");

    const editButton = container.querySelector(
      ".file-details__edit-button"
    ) as HTMLButtonElement;
    await user.click(editButton);

    expect(clickSpy).toHaveBeenCalled();
  });

  it("calls onFileSelect when a file is selected", () => {
    const onFileSelect = jest.fn();
    const { container } = setup({
      gitopsCompatible: false,
      onFileSelect,
    });

    const fileInput = container.querySelector(
      "input[type='file']"
    ) as HTMLInputElement;

    fireEvent.change(fileInput, {
      target: {
        files: [new File(["content"], "config.yaml", { type: "text/yaml" })],
      },
    });

    expect(onFileSelect).toHaveBeenCalled();
  });

  it("renders delete button and calls onDeleteFile on click", async () => {
    const onDeleteFile = jest.fn();
    const { user, container } = setup({ onDeleteFile });

    const deleteButton = container.querySelector(
      ".file-details__delete-button"
    ) as HTMLButtonElement;
    await user.click(deleteButton);

    expect(onDeleteFile).toHaveBeenCalled();
  });

  it("renders progress bar and hides edit/delete when progress is present", () => {
    const { container } = setup({ progress: 0.5 });

    expect(
      container.querySelector(".file-details__progress-bar--uploaded")
    ).toBeInTheDocument();
    expect(container.querySelector(".file-details__edit-button")).toBeNull();
    expect(container.querySelector(".file-details__delete-button")).toBeNull();
  });
});
