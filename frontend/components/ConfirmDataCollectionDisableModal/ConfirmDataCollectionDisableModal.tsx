import React from "react";

import { DATASET_LABEL, HistoricalDataConfigKey } from "interfaces/charts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "confirm-data-collection-disable-modal";

interface IConfirmDataCollectionDisableModalProps {
  scope: "global" | "fleet";
  datasets: HistoricalDataConfigKey[];
  fleetName?: string;
  isUpdating?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmDataCollectionDisableModal = ({
  scope,
  datasets,
  fleetName,
  isUpdating,
  onConfirm,
  onCancel,
}: IConfirmDataCollectionDisableModalProps): JSX.Element => {
  const heading =
    scope === "global"
      ? "You're about to disable data collection across this Fleet deployment."
      : `You're about to disable data collection for fleet "${
          fleetName ?? ""
        }".`;

  return (
    <Modal
      className={baseClass}
      title="Disable data collection"
      onExit={onCancel}
      isContentDisabled={isUpdating}
    >
      <>
        <p>{heading}</p>
        <p>The following dataset(s) will be disabled:</p>
        <ul className={`${baseClass}__dataset-list`}>
          {datasets.map((key) => (
            <li key={key}>
              <strong>{DATASET_LABEL[key]}</strong>
            </li>
          ))}
        </ul>
        <p>
          Previously collected data will be deleted.{" "}
          <strong>This cannot be undone.</strong>
        </p>
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onConfirm} isLoading={isUpdating}>
            Save and disable
          </Button>
          <Button variant="inverse-alert" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default ConfirmDataCollectionDisableModal;
