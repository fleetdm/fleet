import React, { ReactNode, useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Graphic from "components/Graphic";
import Icon from "components/Icon";

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
  /** The text to display on the upload button
   * @default "Upload"
   */
  buttonMessage?: string;
  className?: string;
  /** renders the button to open the file uploader to appear as a button or
   * a link.
   * @default "button"
   */
  buttonType?: "button" | "link";
  /** If provided FileUploader will display this component when the file is
   * selected. This is used for previewing the file before uploading.
   */
  filePreview?: ReactNode;
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
  filePreview,
  className,
  buttonMessage = "Upload",
  buttonType = "button",
  onFileUpload,
}: IFileUploaderProps) => {
  const [isFileSelected, setIsFileSelected] = useState(false);

  const classes = classnames(baseClass, className, {
    [`${baseClass}__file-preview`]: filePreview !== undefined && isFileSelected,
  });
  const buttonVariant = buttonType === "button" ? "brand" : "text-icon";

  const onFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    onFileUpload(files);
    setIsFileSelected(true);

    e.target.value = "";
  };

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
      {isFileSelected && filePreview ? (
        filePreview
      ) : (
        <>
          <div className={`${baseClass}__graphics`}>{renderGraphics()}</div>
          <p className={`${baseClass}__message`}>{message}</p>
          {additionalInfo && (
            <p className={`${baseClass}__additional-info`}>{additionalInfo}</p>
          )}
          <Button
            className={`${baseClass}__upload-button`}
            variant={buttonVariant}
            isLoading={isLoading}
          >
            <label htmlFor="upload-file">
              {buttonType === "link" && <Icon name="upload" />}
              <span>{buttonMessage}</span>
            </label>
          </Button>
          <input
            accept={accept}
            id="upload-file"
            type="file"
            onChange={onFileSelect}
          />
        </>
      )}
    </Card>
  );
};

export default FileUploader;
