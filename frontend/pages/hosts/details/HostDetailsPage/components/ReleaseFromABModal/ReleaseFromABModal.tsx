import React, { useState } from "react";

import Modal from "components/Modal";
import Checkbox from "components/forms/fields/Checkbox";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import mdmAbmAPI from "services/entities/mdm_apple_bm";
import { notify } from "components/ToastNotification";
import getErrorMessage from "./helpers";

type SimpleHost = {
  id: number;
  display_name: string;
};

export interface IReleaseFromABModalProps {
  host: SimpleHost;
  onExit: () => void;
  onRelease: () => void;
}

const baseClass = "release-from-ab-modal";

const ReleaseFromABModal = ({
  host,
  onExit,
  onRelease,
}: IReleaseFromABModalProps) => {
  const [isChecked, setIsChecked] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onModalExit = () => {
    setIsChecked(false);
    setIsSubmitting(false);
    onExit();
  };

  const handleRelease = async () => {
    setIsSubmitting(true);

    try {
      const res = await mdmAbmAPI.releaseHostsFromAB([host.id]);
      if (res.results.length === 0) {
        notify.error(
          "No results were returned from the release request. Please try again.",
          { response: res }
        );
        return;
      }

      if (res.results.length > 1) {
        console.warn(
          "More than one result was returned from the release request. Only the first result will be used."
        );
      }

      const result = res.results[0];

      if (result.status === "failed") {
        if (result.error) {
          notify.error(getErrorMessage(result.error), { response: res });
        } else {
          notify.error(
            "An unknown error occurred while releasing the host from Apple Business.",
            { response: res }
          );
        }
      } else {
        notify.success(
          <p>
            Successfully released <b>{host.display_name}</b> from Apple
            Business.
          </p>
        );
        onRelease();
      }
    } catch (e) {
      notify.error(
        "Couldn't send request to release host from Apple Business. Please try again.",
        { response: e }
      );
    } finally {
      onModalExit();
    }
  };

  return (
    <Modal
      title={"Release from Apple Business"}
      className={baseClass}
      onExit={onModalExit}
    >
      <div>
        <p>
          This removes <b>{host.display_name}</b> from your Apple Business and
          can&apos;t be added back automatically.
        </p>
        <p>This won&apos;t unenroll the host from Fleet.</p>
        <div>
          <p>
            <b>Please check to confirm:</b>
          </p>
          <Checkbox
            className={`${baseClass}__confirm-checkbox`}
            value={isChecked}
            onChange={(value: boolean) => setIsChecked(value)}
            variant="danger"
          >
            I understand this action can&apos;t be undone for{" "}
            <b>{host.display_name}</b>.
          </Checkbox>
        </div>
      </div>
      <ModalFooter
        primaryButtons={
          <>
            <Button variant="inverse-alert" onClick={onModalExit}>
              Cancel
            </Button>
            <Button
              variant="alert"
              disabled={!isChecked || isSubmitting}
              isLoading={isSubmitting}
              onClick={handleRelease}
            >
              Release
            </Button>
          </>
        }
      />
    </Modal>
  );
};

export default ReleaseFromABModal;
