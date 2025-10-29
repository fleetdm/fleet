import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";

const baseClass = "bootstrap-package-uploader";

interface IBootstrapPackageUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
}

const BootstrapPackageUploader = ({
  currentTeamId,
  onUpload,
}: IBootstrapPackageUploaderProps) => {
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
      await mdmAPI.uploadBootstrapPackage(file, currentTeamId);
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
    <div className={baseClass}>
      <p>
        Upload a bootstrap package to install a configuration management tool
        (ex. Munki, Chef, or Puppet) on macOS hosts that automatically enroll to
        Fleet.{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package"
          text="Learn more"
          newTab
        />
      </p>
      <FileUploader
        message="Package (.pkg)"
        graphicName="file-pkg"
        accept=".pkg"
        onFileUpload={onUploadFile}
        isLoading={showLoading}
      />
    </div>
  );
};

export default BootstrapPackageUploader;
