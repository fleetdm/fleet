import React, { useState } from "react";

import FileUploader from "components/FileUploader/FileUploader";
import { notify } from "components/ToastNotification";
import useGitOpsMode from "hooks/useGitOpsMode";

import {
  EULA_PLATFORM_CONFIG,
  EulaPlatform,
  getErrorMessage,
  hasExpectedExtension,
} from "../../helpers";

const baseClass = "eula-uploader";

interface IEulaUploaderProps {
  platform: EulaPlatform;
  onUpload: () => void;
}

const EulaUploader = ({ platform, onUpload }: IEulaUploaderProps) => {
  const [showLoading, setShowLoading] = useState(false);
  const { gitOpsModeEnabled } = useGitOpsMode();
  const config = EULA_PLATFORM_CONFIG[platform];

  const onUploadFile = async (files: FileList | null) => {
    setShowLoading(true);

    if (gitOpsModeEnabled || !files || files.length === 0) {
      setShowLoading(false);
      return;
    }

    const file = files[0];

    // quick exit if the file type is incorrect
    if (!hasExpectedExtension(platform, file.name)) {
      notify.error(config.wrongTypeMessage);
      setShowLoading(false);
      return;
    }
    if (config.sizeLimit && file.size > config.sizeLimit.bytes) {
      notify.error(config.sizeLimit.message);
      setShowLoading(false);
      return;
    }

    try {
      await config.upload(file);
      notify.success("Successfully uploaded.");
      onUpload();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <div className={baseClass}>
      <FileUploader
        graphicName={config.graphicName}
        message={config.uploadMessage}
        onFileUpload={onUploadFile}
        accept={config.extension}
        isLoading={showLoading}
        disabled={showLoading}
        gitopsCompatible
        gitOpsModeEnabled={gitOpsModeEnabled}
      />
    </div>
  );
};

export default EulaUploader;
