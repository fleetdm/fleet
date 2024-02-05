import React, { useContext } from "react";

import { AppContext } from "context/app";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "rerun-script-modal";

interface IRerunScriptModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  onRerun: (scriptId: number) => void;
}

const generateMessageSuffix = (isPremiumTier?: boolean, teamId?: number) => {
  if (!isPremiumTier) {
    return "";
  }
  return teamId ? " assigned to this team" : " with no team";
};

const RerunScriptModal = ({
  scriptName,
  scriptId,
  onCancel,
  onRerun,
}: IRerunScriptModalProps) => {
  const { isPremiumTier, currentTeam } = useContext(AppContext);

  const messageSuffix = generateMessageSuffix(isPremiumTier, currentTeam?.id);

  return (
    <Modal
      className={baseClass}
      title="Rerun Script"
      onExit={onCancel}
      onEnter={() => onRerun(scriptId)}
    >
      <>
        <p>
          This action will rerun script{" "}
          <span className={`${baseClass}__script-name`}>{scriptName}</span> on
          all macOS hosts {messageSuffix}.
        </p>
        <p>This may cause the script to run more than once on some hosts.</p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={() => onRerun(scriptId)}>
            Rerun
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RerunScriptModal;
