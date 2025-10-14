import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import FileUploader from "components/FileUploader";

import { getErrorMessage } from "./helpers";

const baseClass = "script-uploader";

interface IScriptPackageUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
  forModal?: boolean;
}

const ScriptPackageUploader = ({
  currentTeamId,
  onUpload,
  forModal,
}: IScriptPackageUploaderProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showLoading, setShowLoading] = useState(false);

  const onUploadFile = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }

    const file = files[0];

    setShowLoading(true);
    try {
      await scriptAPI.uploadScript(file, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    } finally {
      setShowLoading(false);
    }
  };

  const buttonType = forModal ? "brand-inverse-icon" : undefined;
  const buttonMessage = forModal ? "Choose file" : "Add script";

  return (
    <FileUploader
      className={baseClass}
      graphicName={["file-sh", "file-ps1"]}
      message="Shell (.sh) for macOS and Linux or PowerShell (.ps1) for Windows"
      accept=".sh,.ps1"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
      buttonType={buttonType}
      buttonMessage={buttonMessage}
      gitopsCompatible
    />
  );
};

export default ScriptPackageUploader;
