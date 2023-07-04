import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "file-uploader";

interface IFileUploaderProps {
  icon: IconNames;
  message: string;
  isLoading?: boolean;
  accept?: string;
  className?: string;
  onFileUpload: (files: FileList | null) => void;
}

const FileUploader = ({
  icon,
  message,
  isLoading = false,
  accept,
  className,
  onFileUpload,
}: IFileUploaderProps) => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <Icon name={icon} />
      <p>{message}</p>
      <Button variant="brand" isLoading={isLoading}>
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
    </div>
  );
};

export default FileUploader;
