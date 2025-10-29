import React from "react";

import { IActivityDetails } from "interfaces/activity";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import {
  TargetTitle,
  TargetValue,
} from "../LibrarySoftwareDetailsModal/LibrarySoftwareDetailsModal";

const baseClass = "vpp-details-modal";

interface IVppDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const VppDetailsModal = ({ details, onCancel }: IVppDetailsModalProps) => {
  const { labels_include_any, labels_exclude_any } = details;

  return (
    <Modal
      title="Details"
      width="large"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <DataSet title="Name" value={details.software_title} />
          <DataSet title="App Store ID" value={details.app_store_id} />
          <DataSet
            title="Self-Service"
            value={details.self_service ? "Yes" : "No"}
          />
          <DataSet
            title={
              <TargetTitle
                labelIncludeAny={labels_include_any}
                labelExcludeAny={labels_exclude_any}
              />
            }
            value={
              <TargetValue
                labelIncludeAny={labels_include_any}
                labelExcludeAny={labels_exclude_any}
              />
            }
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default VppDetailsModal;
