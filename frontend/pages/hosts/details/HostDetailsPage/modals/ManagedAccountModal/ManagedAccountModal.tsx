import React from "react";
import { useQuery } from "react-query";

import { IHostManagedAccountPasswordResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "managed-account-modal";

interface IManagedAccountModalProps {
  hostId: number;
  onCancel: () => void;
}

const ManagedAccountModal = ({
  hostId,
  onCancel,
}: IManagedAccountModalProps) => {
  const {
    data: managedAccountData,
    error: managedAccountError,
    isLoading,
  } = useQuery<
    IHostManagedAccountPasswordResponse,
    unknown,
    IHostManagedAccountPasswordResponse["managed_account_password"]
  >(
    ["hostManagedAccountPassword", hostId],
    () => hostAPI.getManagedAccountPassword(hostId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.managed_account_password,
      // prevent caching this sensitive string
      cacheTime: 0,
    }
  );

  return (
    <Modal title="Managed account" onExit={onCancel} className={baseClass}>
      {isLoading && <Spinner />}
      {managedAccountError ? (
        <DataError />
      ) : (
        !isLoading && (
          <>
            <div className={`${baseClass}__username`}>
              <span className={`${baseClass}__label`}>Username</span>
              <span className={`${baseClass}__value`}>_fleetadmin</span>
            </div>
            <InputFieldHiddenContent
              value={managedAccountData?.password ?? ""}
              name="Password"
            />
            <div className="modal-cta-wrap">
              <Button onClick={onCancel}>Close</Button>
            </div>
          </>
        )
      )}
    </Modal>
  );
};

export default ManagedAccountModal;
