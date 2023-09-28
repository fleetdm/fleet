import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import FileUploader from "pages/ManageControlsPage/components/FileUploader";

const baseClass = "script-package-uploader";

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

    // quick exit if the file type is incorrect
    if (!file.name.includes(".pkg")) {
      renderFlash("error", UPLOAD_ERROR_MESSAGES.wrongType.message);
      setShowLoading(false);
      return;
    }

    try {
      await scriptAPI.uploadScript(file, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const errMessage = getErrorMessage(error);
      renderFlash("error", errMessage);
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <FileUploader
      message="Package (.pkg)"
      icon="file-pkg"
      accept=".pkg"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default ScriptPackageUploader;
