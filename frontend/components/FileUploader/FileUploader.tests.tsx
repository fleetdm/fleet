import { fireEvent, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import FileUploader from "./FileUploader";

const render = createCustomRenderer();

describe("FileUploader", () => {
  it("fires onFileUpload with the dropped files", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        onFileUpload={onFileUpload}
      />
    );

    const dropZone = container.querySelector(
      ".file-uploader__wrapper"
    ) as HTMLElement;
    const file = new File(["x"], "thing.pkg", { type: "application/x-pkg" });

    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    });

    expect(onFileUpload).toHaveBeenCalledTimes(1);
    const fileList = onFileUpload.mock.calls[0][0] as FileList;
    expect(fileList[0]).toBe(file);
  });

  it("does not fire onFileUpload when disabled", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        disabled
        onFileUpload={onFileUpload}
      />
    );

    const dropZone = container.querySelector(
      ".file-uploader__wrapper"
    ) as HTMLElement;

    fireEvent.drop(dropZone, {
      dataTransfer: {
        files: [new File(["x"], "thing.pkg", { type: "application/x-pkg" })],
      },
    });

    expect(onFileUpload).not.toHaveBeenCalled();
  });

  it("adds a drag-active class on dragover and clears it on drop", () => {
    const { container } = render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        onFileUpload={noop}
      />
    );

    const card = container.querySelector(".file-uploader") as HTMLElement;
    const dropZone = container.querySelector(
      ".file-uploader__wrapper"
    ) as HTMLElement;

    fireEvent.dragOver(dropZone);
    expect(card.className).toContain("file-uploader__drag-active");

    fireEvent.drop(dropZone, {
      dataTransfer: {
        files: [new File(["x"], "thing.pkg", { type: "application/x-pkg" })],
      },
    });
    expect(card.className).not.toContain("file-uploader__drag-active");
  });

  it("still renders the message so existing click-to-upload callers are untouched", () => {
    render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        onFileUpload={noop}
      />
    );
    expect(screen.getByText("drop a package")).toBeInTheDocument();
  });
});
