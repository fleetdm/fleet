import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import FileUploader from "components/FileUploader";

import { getErrorMessage } from "./helpers";

const baseClass = "script-uploader";

interface IScriptPackageUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
}

const ScriptPackageUploader = ({
  currentTeamId,
  onUpload,
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

  return (
    <FileUploader
      className={baseClass}
      graphicName={["file-sh", "file-ps1"]}
      message="Shell (.sh) for macOS and Linux or PowerShell (.ps1) for Windows"
      additionalInfo="Script will run with “#!/bin/sh”, “#!/bin/zsh”, or “#!/bin/bash” on macOS and Linux."
      accept=".sh,.ps1"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default ScriptPackageUploader;
