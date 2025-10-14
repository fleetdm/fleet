import React from "react";

import Modal from "components/Modal";
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
            onUpload={onSubmit}
            forModal
          />
        </div>
      </>
    </Modal>
  );
};

export default ScriptUploadModal;
