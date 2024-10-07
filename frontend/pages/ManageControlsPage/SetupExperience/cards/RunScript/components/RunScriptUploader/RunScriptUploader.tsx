import React, { useContext, useState } from "react";
import classnames from "classnames";

import { NotificationContext } from "context/notification";
import FileUploader from "components/FileUploader";

const baseClass = "run-script-uploader";

interface IRunScriptUploaderProps {
  onUpload: () => void;
  className?: string;
}

const RunScriptUploader = ({
  onUpload,
  className,
}: IRunScriptUploaderProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showLoading, setShowLoading] = useState(false);

  const classNames = classnames(baseClass, className);

  const onUploadFile = async (files: FileList | null) => {};

  return (
    <FileUploader
      className={classNames}
      message="Shell (.sh) for macOS"
      graphicName="file-sh"
      accept=".sh"
      buttonMessage="Upload"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default RunScriptUploader;
