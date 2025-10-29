import React, { useContext, useState } from "react";
import classnames from "classnames";

import mdmAPI from "services/entities/mdm";

import { NotificationContext } from "context/notification";
import FileUploader from "components/FileUploader";
import { getErrorReason } from "interfaces/errors";

const baseClass = "setup-experience-script-uploader";

interface ISetupExperienceScriptUploaderProps {
  currentTeamId: number;
  hasManualAgentInstall: boolean;
  onUpload: () => void;
  className?: string;
}

const SetupExperienceScriptUploader = ({
  currentTeamId,
  hasManualAgentInstall,
  onUpload,
  className,
}: ISetupExperienceScriptUploaderProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showLoading, setShowLoading] = useState(false);

  const classNames = classnames(baseClass, className);

  const onUploadFile = async (files: FileList | null) => {
    setShowLoading(true);

    if (!files || files.length === 0) {
      setShowLoading(false);
      return;
    }

    const file = files[0];

    try {
      await mdmAPI.uploadSetupExperienceScript(file, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      // TODO: what errors?
      renderFlash("error", getErrorReason(e));
    }

    setShowLoading(false);
  };

  const manuallyInstallTooltipText = (
    <>
      Disabled because you manually install Fleet&apos;s agent (
      <b>Bootstrap package {">"} Advanced options</b>). Use your bootstrap
      package to install software during the setup experience.
    </>
  );

  return (
    <FileUploader
      className={classNames}
      message="Shell (.sh) for macOS"
      graphicName="file-sh"
      accept=".sh"
      buttonMessage="Upload"
      onFileUpload={onUploadFile}
      isLoading={showLoading}
      disabled={hasManualAgentInstall}
      buttonTooltip={hasManualAgentInstall && manuallyInstallTooltipText}
      gitopsCompatible
    />
  );
};

export default SetupExperienceScriptUploader;
