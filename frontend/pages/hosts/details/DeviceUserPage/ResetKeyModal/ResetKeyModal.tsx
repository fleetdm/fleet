import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import mdmAPI from "services/entities/mdm";
import { useQuery } from "react-query";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

interface IResetKeyModalProps {
  onClose: () => void;
  deviceAuthToken: string;
}

const baseClass = "reset-key-modal";

const ResetKeyModal = ({
  onClose,
  deviceAuthToken,
}: IResetKeyModalProps): JSX.Element => {
  const { isLoading: isLoadingResetDEKey, error: errorResetDEKey } = useQuery(
    ["resetDEkey", deviceAuthToken],
    () => mdmAPI.resetEncryptionKey(deviceAuthToken),
    { refetchOnWindowFocus: false }
  );

  const renderModalBody = () => {
    if (isLoadingResetDEKey) {
      return <Spinner />;
    }
    if (errorResetDEKey) {
      return <DataError />;
    }

    return (
      <div>
        <ol>
          <li>
            Wait 30 seconds for the <b>Reset disk encryption key</b> pop up to
            open.
          </li>
          <li>
            In the popup, enter the password you use to login to your Mac.
          </li>
          <li>
            Close this window and select <b>Refetch</b> on your My device page.
            This tells your organization that you reset your key.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onClose} variant="brand">
            Done
          </Button>
        </div>
      </div>
    );
  };
  return (
    <Modal
      title="Reset key"
      onExit={onClose}
      className={baseClass}
      width="medium"
    >
      {renderModalBody()}
    </Modal>
  );
};

export default ResetKeyModal;
