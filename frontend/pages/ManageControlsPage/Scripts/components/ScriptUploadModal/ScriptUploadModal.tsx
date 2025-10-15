import React, { useContext, useState } from "react";
import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { getErrorMessage } from "./helpers";
import ScriptUploader from "../ScriptUploader";

const baseClass = "script-upload-modal";

interface IScriptUploadModal {
  onExit: () => void;
  onSubmit: () => void;
  currentTeamId: number;
}

const ScriptUploadModal = ({
  onSubmit,
  onExit,
  currentTeamId,
}: IScriptUploadModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [showLoading, setShowLoading] = useState(false);

  const onUploadFile = async () => {
    if (!selectedFile) {
      return;
    }
    setShowLoading(true);
    try {
      await scriptAPI.uploadScript(selectedFile, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onSubmit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    } finally {
      setShowLoading(false);
    }
  };

  const additionalInfo =
    selectedFile && selectedFile.name.match(/\.sh$/)
      ? 'On macOS and Linux, script will run according to the interpreter specified in the first line: "#!/bin/sh", "#!/bin/zsh", or "#!/bin/bash"'
      : undefined;

  return (
    <Modal
      title="Add script"
      onExit={onExit}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__content`}>
          <ScriptUploader
            currentTeamId={currentTeamId}
            onFileSelected={(file) => setSelectedFile(file)}
            selectedFile={selectedFile}
            forModal
          />
        </div>
        {additionalInfo && (
          <p className={`${baseClass}__additional-info`}>{additionalInfo}</p>
        )}
        <div className="modal-cta-wrap">
          <Button
            onClick={onUploadFile}
            disabled={!selectedFile || showLoading}
            isLoading={showLoading}
          >
            Add Script
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default ScriptUploadModal;
