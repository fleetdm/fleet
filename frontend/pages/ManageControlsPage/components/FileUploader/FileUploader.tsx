import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "file-uploader";

interface IFileUploaderProps {
  icon: IconNames;
  message: string;
  isLoading?: boolean;
  onFileUpload: (files: FileList | null) => void;
}

const FileUploader = ({
  icon,
  message,
  isLoading = false,
  onFileUpload,
}: IFileUploaderProps) => {
  return (
    <div className={baseClass}>
      <Icon name={icon} />
      <p>{message}</p>
      <Button isLoading={isLoading}>
        <label htmlFor="upload-profile">Upload</label>
      </Button>
      <input
        accept=".mobileconfig,application/x-apple-aspen-config"
        id="upload-profile"
        type="file"
        onChange={(e) => onFileUpload(e.target.files)}
      />
    </div>
  );
};

export default FileUploader;
