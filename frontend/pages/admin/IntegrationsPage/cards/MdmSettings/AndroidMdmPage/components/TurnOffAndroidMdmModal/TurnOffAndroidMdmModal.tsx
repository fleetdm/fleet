import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQueryClient } from "react-query";

import PATHS from "router/paths";
import mdmAndroidAPI from "services/entities/mdm_android";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import refetchConfigUntil from "../../helpers";

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
  const { setConfig } = useContext(AppContext);
  const queryClient = useQueryClient();

  const [isDeleting, setIsDeleting] = useState(false);

  const onClickConfirm = useCallback(async () => {
    setIsDeleting(true);
    try {
      await mdmAndroidAPI.turnOffAndroidMdm();
      // Mirror the turn-on flow: pull fresh config, update AppContext so the parent MDM page's
      // card flips immediately on redirect, and invalidate the React Query cache for downstream consumers.
      const freshConfig = await refetchConfigUntil(
        (cfg) => !cfg.mdm.android_enabled_and_configured
      );
      setConfig(freshConfig);
      await queryClient.invalidateQueries(["config"]);
      renderFlash("success", "Android MDM turned off successfully.", {
        persistOnPageChange: true,
      });
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    } catch (e) {
      onExit();
      renderFlash("error", "Couldn't turn off Android MDM. Please try again.");
    }
  }, [onExit, queryClient, renderFlash, router, setConfig]);

  return (
    <Modal title="Turn off Android MDM" className={baseClass} onExit={onExit}>
      <p>
        If you want to use MDM features again, you&apos;ll have to reconnect
        Android Enterprise.
      </p>
      <p>
        End users will lose access to organization resources and all data in
        their Android work partition.
      </p>
      <div className="modal-cta-wrap">
        <Button
          variant="alert"
          isLoading={isDeleting}
          disabled={isDeleting}
          onClick={onClickConfirm}
        >
          Turn off
        </Button>
        <Button variant="inverse-alert" disabled={isDeleting} onClick={onExit}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default TurnOffAndroidMdmModal;
