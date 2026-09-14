import React, { useCallback, useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import mdmAbmAPI from "services/entities/mdm_apple_bm";

const baseClass = "delete-abm-modal";

interface IDeleteAbmModalProps {
  tokenOrgName: string;
  tokenId: number;
  /** Whether the token being deleted is the default one; deleting a
   * non-default token doesn't affect the default, so no extra copy. */
  tokenIsDefault: boolean;
  /** Count of AB tokens before this deletion, used to pick the copy about
   * what happens to the default token afterwards. */
  tokensCount: number;
  onCancel: () => void;
  onDeletedToken: () => void;
}

const DeleteAbmModal = ({
  tokenOrgName,
  tokenId,
  tokenIsDefault,
  tokensCount,
  onCancel,
  onDeletedToken,
}: IDeleteAbmModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = useCallback(async () => {
    setIsDeleting(true);

    try {
      await mdmAbmAPI.deleteToken(tokenId);
      notify.success("Deleted successfully.");
      onDeletedToken();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      notify.error("Couldn’t disable automatic enrollment. Please try again.", {
        response: e,
      });
      onCancel();
    }
  }, [onCancel, onDeletedToken, tokenId]);

  return (
    <Modal
      title="Delete AB"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <p>
        New hosts purchased in <b>{tokenOrgName}</b> won&apos;t automatically
        enroll to Fleet.
        {tokenIsDefault && tokensCount === 2 && (
          <> Your remaining token will become the default automatically.</>
        )}
        {tokenIsDefault && tokensCount > 2 && (
          <>
            {" "}
            Manual enrollments may not be able to sign into Managed Apple IDs
            until you set a new default.
          </>
        )}
      </p>
      <p>
        If you want to re-enable automatic enrollment, you&apos;ll have to
        upload a new AB token.
      </p>

      <div className="modal-cta-wrap">
        <Button
          type="button"
          variant="alert"
          onClick={onDeleteToken}
          disabled={isDeleting}
          isLoading={isDeleting}
        >
          Delete
        </Button>
        <Button onClick={onCancel} disabled={isDeleting} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteAbmModal;
