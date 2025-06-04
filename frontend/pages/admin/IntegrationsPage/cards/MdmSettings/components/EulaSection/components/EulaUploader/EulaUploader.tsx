import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import FileUploader from "components/FileUploader/FileUploader";
import CustomLink from "components/CustomLink";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";

const baseClass = "eula-uploader";

interface IEulaUploaderProps {
  onUpload: () => void;
}

const EulaUploader = ({ onUpload }: IEulaUploaderProps) => {
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
    if (!file.name.includes(".pdf")) {
      renderFlash("error", UPLOAD_ERROR_MESSAGES.wrongType.message);
      setShowLoading(false);
      return;
    }

    try {
      await mdmAPI.uploadEULA(file);
      renderFlash("success", "Successfully updated end user authentication!");
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
    <div className={baseClass}>
      <p>
        Require end users to agree to a EULA when they first setup their new
        macOS hosts.{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/end-user-license-agreement"
          text="Learn more"
          newTab
        />
      </p>
      <FileUploader
        graphicName="file-pdf"
        message="PDF (.pdf)"
        onFileUpload={onUploadFile}
        accept=".pdf"
        isLoading={showLoading}
      />
    </div>
  );
};

export default EulaUploader;
