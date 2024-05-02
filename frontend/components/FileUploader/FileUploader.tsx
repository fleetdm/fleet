import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Graphic from "components/Graphic";

const baseClass = "file-uploader";

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
  accept,
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
      <input
        accept={accept}
        id="upload-file"
        type="file"
        onChange={(e) => {
          onFileUpload(e.target.files);
          e.target.value = "";
        }}
      />
    </Card>
  );
};

export default FileUploader;
