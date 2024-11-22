import React, { useState, useRef } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Icon from "components/Icon";
import Graphic from "components/Graphic";
import FileDetails from "components/FileDetails";

const baseClass = "file-uploader";

export type ISupportedGraphicNames = Extract<
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
  | "file-vpp"
  | "file-crt"
>;

interface IFileUploaderProps {
  graphicName: ISupportedGraphicNames | ISupportedGraphicNames[];
  message: string;
  additionalInfo?: string;
  /** Controls the loading spinner on the upload button */
  isLoading?: boolean;
  /** Disables the upload button */
  disabled?: boolean;
  /** A comma separated string of one or more file types accepted to upload.
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
  onFileUpload: (files: FileList | null) => void;
  /** renders the current file with the edit pencil button */
  canEdit?: boolean;
  fileDetails?: {
    name: string;
    platform?: string;
  };
}

/**
 * A component that encapsulates the UI for uploading a file and a file selected.
 */
export const FileUploader = ({
  graphicName: graphicNames,
  message,
  additionalInfo,
  isLoading = false,
  disabled = false,
  accept,
  className,
  buttonMessage = "Upload",
  buttonType = "button",
  onFileUpload,
  canEdit = false,
  fileDetails,
}: IFileUploaderProps) => {
  const [isFileSelected, setIsFileSelected] = useState(!!fileDetails);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const classes = classnames(baseClass, className, {
    [`${baseClass}__file-preview`]: isFileSelected,
  });
  const buttonVariant = buttonType === "button" ? "brand" : "text-icon";

  const triggerFileInput = () => {
    fileInputRef.current?.click();
  };

  const onFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    onFileUpload(files);
    setIsFileSelected(true);

    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      triggerFileInput();
    }
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

  const renderFileUploader = () => {
    return (
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
          disabled={disabled}
          customOnKeyDown={handleKeyDown}
          tabIndex={0}
        >
          <label htmlFor="upload-file">
            {buttonType === "link" && <Icon name="upload" />}
            <span>{buttonMessage}</span>
          </label>
        </Button>
        <input
          ref={fileInputRef}
          accept={accept}
          id="upload-file"
          type="file"
          onChange={onFileSelect}
        />
      </>
    );
  };

  return (
    <Card color="gray" className={classes}>
      {isFileSelected && fileDetails ? (
        <FileDetails
          graphicNames={graphicNames}
          fileDetails={fileDetails}
          canEdit={canEdit}
          onFileSelect={onFileSelect}
          accept={accept}
        />
      ) : (
        renderFileUploader()
      )}
    </Card>
  );
};

export default FileUploader;
