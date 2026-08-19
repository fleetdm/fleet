import React, { useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { notify } from "components/ToastNotification";
import mdmAPI from "services/entities/mdm";

import FileUploader from "components/FileUploader";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";

interface IBootstrapPackageUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
}

const BootstrapPackageUploader = ({
  currentTeamId,
  onUpload,
}: IBootstrapPackageUploaderProps) => {
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
      notify.error(UPLOAD_ERROR_MESSAGES.wrongType.message);
      setShowLoading(false);
      return;
    }

    try {
      await mdmAPI.uploadBootstrapPackage(file, currentTeamId);
      notify.success("Successfully uploaded.");
      onUpload();
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const errMessage = getErrorMessage(error);
      notify.error(errMessage, { response: e });
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <FileUploader
      message="Package (.pkg)"
      graphicName="file-pkg"
      accept=".pkg"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
    />
  );
};

export default BootstrapPackageUploader;
