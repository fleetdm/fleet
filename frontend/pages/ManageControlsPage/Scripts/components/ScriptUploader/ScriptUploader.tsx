import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
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
      const error = e as AxiosResponse<IApiError>;
      const apiErrMessage = getErrorMessage(error);
      const renderErrMessage = apiErrMessage.includes(
        "File type not supported. Only .sh and .ps1 file type is allowed."
      )
        ? // per https://github.com/fleetdm/fleet/issues/14752#issuecomment-1809927441
          "The file should be .sh or .ps1 file."
        : apiErrMessage;
      renderFlash("error", `Couldn't upload. ${renderErrMessage}`);
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <FileUploader
      className={baseClass}
      graphicName={["file-sh", "file-ps1"]}
      message="Shell (.sh) for macOS and Linux or PowerShell (.ps1) for Windows"
      additionalInfo="Script will run with “#!/bin/sh” or “#!/bin/zsh” on macOS and Linux."
      accept=".sh,.ps1"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default ScriptPackageUploader;
