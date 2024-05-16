import React, { useState } from "react";
import classnames from "classnames";
import { FileUploader as FileUploaderDragDrop } from "react-drag-drop-files";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Graphic from "components/Graphic";

const baseClass = "file-uploader";

const fileTypes = ["SH", "PS1"];

type ISupportedGraphicNames = Extract<
  GraphicNames,
  | "file-configuration-profile"
  | "file-sh"
  | "file-ps1"
  | "file-py"
  | "file-script"
  | "file-pdf"
  | "file-pkg"
  | "file-p7m"
  | "file-pem"
>;

interface IFileUploaderProps {
  graphicName: ISupportedGraphicNames | ISupportedGraphicNames[];
  message: string;
  additionalInfo?: string;
  /** Controls the loading spinner on the upload button */
  isLoading?: boolean;
  /** A comma seperated string of one or more file types accepted to upload.
   * This is the same as the html accept attribute.
   * https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/accept
   */
  accept?: string;
  /** The text to display on the upload button */
  buttonMessage?: string;
  className?: string;
  onFileUpload: (files: FileList | null) => void;
}

/**
 * A component that encapsulates the UI for uploading a file.
 */
const FileUploader = ({
  graphicName: graphicNames,
  message,
  additionalInfo,
  isLoading = false,
  buttonMessage = "Upload",
  className,
  onFileUpload,
}: IFileUploaderProps) => {
  const classes = classnames(baseClass, className);

  const renderGraphics = () => {
    const graphicNamesArr =
      typeof graphicNames === "string" ? [graphicNames] : graphicNames;
    return graphicNamesArr.map((graphicName) => (
      <Graphic
        key={`${graphicName}-graphic`}
        className={`${baseClass}__graphic`}
        name={graphicName}
      />
    ));
  };

  function RenderDragDrop() {
    const handleChange = (files: FileList) => {
      console.log("files: ", files);
      onFileUpload(files);
    };
    return (
      <FileUploaderDragDrop
        handleChange={handleChange}
        types={fileTypes}
        name="file"
        type={fileTypes}
        classes="file-uploader__drag-drop"
        label=" "
        multiple
      />
    );
  }

  return (
    <Card color="gray" className={classes}>
      <div className={`${baseClass}__graphics`}>{renderGraphics()}</div>
      <p className={`${baseClass}__message`}>{message}</p>
      {additionalInfo && (
        <p className={`${baseClass}__additional-info`}>{additionalInfo}</p>
      )}
      <Button
        className={`${baseClass}__upload-button`}
        variant="brand"
        isLoading={isLoading}
      >
        <label htmlFor="upload-file">{buttonMessage}</label>
      </Button>
      <RenderDragDrop />
    </Card>
  );
};

export default FileUploader;
