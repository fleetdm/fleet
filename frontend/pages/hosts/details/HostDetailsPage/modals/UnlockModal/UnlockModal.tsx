import React, { useContext } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import hostAPI, { IUnlockHostResponse } from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "unlock-modal";

interface IUnlockModalProps {
  id: number;
  platform: string;
  hostName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const UnlockModal = ({
  id,
  platform,
  hostName,
  onSuccess,
  onClose,
}: IUnlockModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUnlocking, setIsUnlocking] = React.useState(false);

  const {
    data: macUnlockData,
    isLoading: macIsLoading,
    isError: macIsError,
  } = useQuery<IUnlockHostResponse, AxiosError>(
    ["mac-unlock-pin", id],
    () => hostAPI.unlockHost(id),
    {
      enabled: platform === "darwin",
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
      retry: false,
    }
  );

  const onUnlock = async () => {
    setIsUnlocking(true);
    try {
      await hostAPI.unlockHost(id);
      onSuccess();
      renderFlash(
        "success",
        "Unlocking host or will unlock when it comes online."
      );
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onClose();
    setIsUnlocking(false);
  };

  const renderModalContent = () => {
    if (platform === "darwin") {
      if (macIsLoading) return <Spinner />;
      if (macIsError) return <DataError />;

      if (!macUnlockData) return null;

      return (
        <>
          {/* TODO: replace with DataSet component */}
          <p>
            When the host is returned, use the 6-digit PIN to unlock the host.
          </p>
          <div className={`${baseClass}__pin`}>
            <b>PIN</b>
            <span>{macUnlockData.unlock_pin}</span>
          </div>
        </>
      );
    }

    return (
      <>
        <p>
          Are you sure you&apos;re ready to unlock <b>{hostName}</b>?
        </p>
      </>
    );
  };

  const renderModalButtons = () => {
    if (platform === "darwin") {
      return (
        <>
          <Button type="button" onClick={onClose} variant="brand">
            Done
          </Button>
        </>
      );
    }

    return (
      <>
        <Button
          type="button"
          onClick={onUnlock}
          variant="brand"
          className="delete-loading"
          isLoading={isUnlocking}
        >
          Unlock
        </Button>
        <Button onClick={onClose} variant="inverse">
          Cancel
        </Button>
      </>
    );
  };

  return (
    <Modal className={baseClass} title="Unlock host" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          {renderModalContent()}
        </div>

        <div className="modal-cta-wrap">{renderModalButtons()}</div>
      </>
    </Modal>
  );
};

export default UnlockModal;
