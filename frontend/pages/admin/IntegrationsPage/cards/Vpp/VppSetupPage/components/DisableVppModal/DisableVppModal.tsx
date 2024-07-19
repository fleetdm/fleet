import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import paths from "router/paths";

import mdmAppleAPI from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "diable-vpp-modal";

interface IDisableVppModalProps {
  router: InjectedRouter;
  onExit: () => void;
}

const DisableVppModal = ({ router, onExit }: IDisableVppModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDisabling, setIsDisabling] = useState(false);

  const onDisableVpp = async () => {
    setIsDisabling(true);
    try {
      await mdmAppleAPI.disableVpp();
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) disabled successfully."
      );
      router.push(paths.ADMIN_INTEGRATIONS_VPP);
    } catch {
      renderFlash(
        "error",
        "Couldn't disable Volume Purchasing Program (VPP). Please try again."
      );
    }

    onExit();
  };

  return (
    <Modal
      title="Disable Volume Purchasing Program (VPP)"
      onExit={onExit}
      className={baseClass}
    >
      <>
        <p>
          Apps purchased in Apple Business Manager won&apos;t appear in Fleet.
          Apps won&apos;t be uninstalled from hosts. If you want to enable VPP
          integration again, you&apos;ll have to upload a new content token.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDisableVpp}
            isLoading={isDisabling}
          >
            Disable
          </Button>
          <Button onClick={onExit} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DisableVppModal;
