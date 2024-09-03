import React, { ReactNode, useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Icon from "components/Icon";
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
  | "file-vpp"
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

  const classes = classnames(baseClass, className, {
    [`${baseClass}__file-preview`]: isFileSelected,
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

  const renderFileDetails = () => (
    <div className={`${baseClass}__file`}>
      <div className={`${baseClass}__file-info`}>
        <Graphic
          name={
            typeof graphicNames === "string" ? graphicNames : graphicNames[0]
          }
        />
        <div className={`${baseClass}__file-content`}>
          <div className={`${baseClass}__file-name`}>{fileDetails?.name}</div>
          {fileDetails?.platform && (
            <div className={`${baseClass}__file-platform`}>
              {fileDetails.platform}
            </div>
          )}
        </div>
      </div>
      {canEdit && (
        <div className={`${baseClass}__file-edit`}>
          <Button className={`${baseClass}__edit-button`} variant="icon">
            <label htmlFor="edit-file">
              <Icon name="pencil" color="ui-fleet-black-75" />
            </label>
          </Button>
          <input
            accept={accept}
            id="edit-file"
            type="file"
            onChange={onFileSelect}
          />
        </div>
      )}
    </div>
  );

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
    );
  };

  return (
    <Card color="gray" className={classes}>
      {isFileSelected && fileDetails
        ? renderFileDetails()
        : renderFileUploader()}
    </Card>
  );
};

export default FileUploader;
