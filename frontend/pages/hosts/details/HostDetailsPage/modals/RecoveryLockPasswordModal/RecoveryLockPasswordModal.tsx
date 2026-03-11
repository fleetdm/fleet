import React from "react";
import { useQuery } from "react-query";

import { getErrorReason } from "interfaces/errors";
import { IHostRecoveryLockPasswordResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

const baseClass = "recovery-lock-password-modal";

interface IRecoveryLockPasswordModalProps {
  hostId: number;
  onCancel: () => void;
}

const RecoveryLockPasswordModal = ({
  hostId,
  onCancel,
}: IRecoveryLockPasswordModalProps) => {
  const {
    data: recoveryLockPassword,
    error: recoveryLockPasswordError,
    isLoading,
  } = useQuery<IHostRecoveryLockPasswordResponse, unknown, string>(
    ["hostRecoveryLockPassword", hostId],
    () => hostAPI.getRecoveryLockPassword(hostId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.recovery_lock_password.password,
      // prevent caching this sensitive string
      cacheTime: 0,
    }
  );

  return (
    <Modal
      title="Recovery Lock password"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {isLoading && <Spinner />}
      {recoveryLockPasswordError ? (
        <DataError
          description={getErrorReason(recoveryLockPasswordError) || undefined}
        />
      ) : (
        !isLoading && (
          <>
            <InputFieldHiddenContent value={recoveryLockPassword ?? ""} />
            <p>
              Use this to unlock and regain access to the host if the end user
              forgets their local password.{" "}
              <CustomLink
                newTab
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/startup-security-macos`}
                text="Learn more"
              />
            </p>
            <div className="modal-cta-wrap">
              <Button onClick={onCancel}>Done</Button>
            </div>
          </>
        )
      )}
    </Modal>
  );
};

export default RecoveryLockPasswordModal;
