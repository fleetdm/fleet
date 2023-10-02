import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import FileUploader from "pages/ManageControlsPage/components/FileUploader";
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
    setShowLoading(true);

    if (!files || files.length === 0) {
      setShowLoading(false);
      return;
    }

    const file = files[0];

    try {
      await scriptAPI.uploadScript(file, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      renderFlash("error", `Couldn't upload. ${getErrorMessage(error)}`);
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <FileUploader
      className={baseClass}
      icon="file-bash"
      message="Script (.sh)"
      additionalInfo="Script will run with “#!/bin/sh”."
      accept=".sh"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default ScriptPackageUploader;
