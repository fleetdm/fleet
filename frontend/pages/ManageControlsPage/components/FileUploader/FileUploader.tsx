import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Card from "components/Card";

const baseClass = "file-uploader";

interface IFileUploaderProps {
  icon: IconNames;
  message: string;
  additionalInfo?: string;
  isLoading?: boolean;
  accept?: string;
  className?: string;
  onFileUpload: (files: FileList | null) => void;
}

const FileUploader = ({
  icon,
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
      <Icon name={icon} />
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
