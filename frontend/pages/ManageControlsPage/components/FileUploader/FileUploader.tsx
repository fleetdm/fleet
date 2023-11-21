import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Card from "components/Card";
import { GraphicNames } from "components/graphics";
import Graphic from "components/Graphic";

const baseClass = "file-uploader";

interface IFileUploaderProps {
  graphicName: GraphicNames;
  message: string;
  additionalInfo?: string;
  isLoading?: boolean;
  /** A comma seperated string of one or more file types accepted to upload.
   * This is the same as the html accept attribute.
   * https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/accept
   */
  accept?: string;
  className?: string;
  onFileUpload: (files: FileList | null) => void;
}

const FileUploader = ({
  graphicName,
  message,
  additionalInfo,
  isLoading = false,
  accept,
  className,
  onFileUpload,
}: IFileUploaderProps) => {
  const classes = classnames(baseClass, className);

  return (
    <Card color="gray" className={classes}>
      <Graphic name={graphicName} />
      <p className={`${baseClass}__message`}>{message}</p>
      <p className={`${baseClass}__additional-info`}>{additionalInfo}</p>
      <Button
        className={`${baseClass}__upload-button`}
        variant="brand"
        isLoading={isLoading}
      >
        <label htmlFor="upload-profile">Upload</label>
      </Button>
      <input
        accept={accept}
        id="upload-profile"
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
