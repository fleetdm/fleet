import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import mdmAndroidAPI from "services/entities/mdm_android";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "turn-off-android-mdm-modal";

interface ITurnOffAndroidMdmModalProps {
  onExit: () => void;
  router: InjectedRouter;
}

const TurnOffAndroidMdmModal = ({
  onExit,
  router,
}: ITurnOffAndroidMdmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onTurnOffAndroidMdm = async () => {
    try {
      await mdmAndroidAPI.turnOffAndroidMdm();
      renderFlash("success", "Android MDM turned off successfully.", {
        persistOnPageChange: true,
      });
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    } catch (e) {
      onExit();
      renderFlash("error", "Couldn't turn off Android MDM. Please try again.");
    }
  };

  return (
    <Modal title="Turn off Android MDM" className={baseClass} onExit={onExit}>
      <>
        <p>
          If you want to use MDM features again, you&apos;ll have to reconnect
          Android Enterprise.
        </p>
        <p>
          End users will lose access to organization resources and all data in
          their Android work partition.
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onTurnOffAndroidMdm} variant="alert">
            Turn off
          </Button>
          <Button onClick={onExit} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default TurnOffAndroidMdmModal;
