import React from "react";

import FileUploader from "components/FileUploader";
import { getFileDetails } from "utilities/file/fileUtils";

const baseClass = "script-uploader";

interface IScriptPackageUploaderProps {
  onFileSelected?: (file: File) => void;
  selectedFile?: File | null;
  forModal?: boolean;
  onButtonClick?: () => void;
}

const ScriptPackageUploader = ({
  forModal,
  onFileSelected,
  selectedFile,
  onButtonClick,
}: IScriptPackageUploaderProps) => {
  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      onFileSelected?.(files[0]);
    }
  };

  const buttonType = forModal ? "brand-inverse-icon" : undefined;
  const buttonMessage = forModal ? "Choose file" : "Add script";
  const extension = selectedFile?.name.match(/(sh|ps1)$/i)?.[1];
  const graphicName = extension === "ps1" ? "file-ps1" : "file-sh";

  return (
    <FileUploader
      className={baseClass}
      graphicName={[graphicName]}
      message="Shell (.sh) for macOS and Linux or PowerShell (.ps1) for Windows"
      title="Upload script"
      accept=".sh,.ps1"
      onFileUpload={onFileSelect}
      fileDetails={selectedFile ? getFileDetails(selectedFile) : undefined}
      buttonType={buttonType}
      buttonMessage={buttonMessage}
      gitopsCompatible
      onButtonClick={onButtonClick}
    />
  );
};

export default ScriptPackageUploader;
