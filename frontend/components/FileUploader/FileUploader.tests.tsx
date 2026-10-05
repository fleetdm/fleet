import { fireEvent, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import FileUploader from "./FileUploader";

const render = createCustomRenderer();

const findDropZone = (container: HTMLElement) =>
  container.querySelector(".file-uploader__wrapper") as HTMLElement;

const makeFile = (name: string, type = "application/octet-stream") =>
  new File(["x"], name, { type });

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

    const file = makeFile("thing.pkg");
    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [file] },
    });

    expect(onFileUpload).toHaveBeenCalledTimes(1);
    expect((onFileUpload.mock.calls[0][0] as FileList)[0]).toBe(file);
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

    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [makeFile("thing.pkg")] },
    });

    expect(onFileUpload).not.toHaveBeenCalled();
  });

  it("does not fire onFileUpload while isLoading", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        isLoading
        onFileUpload={onFileUpload}
      />
    );

    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [makeFile("thing.pkg")] },
    });

    expect(onFileUpload).not.toHaveBeenCalled();
  });

  it("does not fire onFileUpload when GitOps mode suppresses the uploader", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pkg"
        message="drop a package"
        gitopsCompatible
        gitOpsModeEnabled
        onFileUpload={onFileUpload}
      />
    );

    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [makeFile("thing.pkg")] },
    });

    expect(onFileUpload).not.toHaveBeenCalled();
  });

  it("rejects a drop when any file falls outside the accept filter", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pem"
        message="drop a cert"
        accept=".pem"
        onFileUpload={onFileUpload}
      />
    );

    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [makeFile("not-a-cert.txt")] },
    });

    expect(onFileUpload).not.toHaveBeenCalled();
  });

  it("accepts a drop whose file matches the accept filter", () => {
    const onFileUpload = jest.fn();
    const { container } = render(
      <FileUploader
        graphicName="file-pem"
        message="drop a cert"
        accept=".pem"
        onFileUpload={onFileUpload}
      />
    );

    fireEvent.drop(findDropZone(container), {
      dataTransfer: { files: [makeFile("cert.pem")] },
    });

    expect(onFileUpload).toHaveBeenCalledTimes(1);
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
