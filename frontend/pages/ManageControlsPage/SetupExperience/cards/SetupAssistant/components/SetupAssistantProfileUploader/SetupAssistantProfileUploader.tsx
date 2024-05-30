import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import FileUploader from "components/FileUploader";

import { getErrorMessage } from "./helpers";

const baseClass = "setup-assistant-profile-uploader";

interface ISetupAssistantProfileUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
}

const SetupAssistantProfileUploader = ({
  currentTeamId,
  onUpload,
}: ISetupAssistantProfileUploaderProps) => {
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
      await mdmAPI.uploadSetupEnrollmentProfile(file, currentTeamId);
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
      message="Automatic enrollment profile (.json)"
      graphicName="file-configuration-profile"
      accept=".json"
      buttonMessage="Add profile"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
      className={baseClass}
    />
  );
};

export default SetupAssistantProfileUploader;
