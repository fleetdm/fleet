import React from "react";

import { IActivityDetails } from "interfaces/activity";
import { ILabelSoftwareTitle } from "interfaces/label";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "software-details-modal";

interface ITargetValueProps {
  labelIncludeAny?: ILabelSoftwareTitle[];
  labelExcludeAny?: ILabelSoftwareTitle[];
}

const TargetValue = ({
  labelIncludeAny,
  labelExcludeAny,
}: ITargetValueProps) => {
  if (!labelIncludeAny && !labelExcludeAny) {
    return <>All hosts</>;
  }

  let valueText = "";
  let labels: ILabelSoftwareTitle[] = [];
  if (labelIncludeAny) {
    valueText = "Custom - include any label";
    labels = labelIncludeAny;
  } else if (labelExcludeAny) {
    valueText = "Custom - exclude any label";
    labels = labelExcludeAny;
  }

  return (
    <TooltipWrapper
      tipContent={labels.map((label) => (
        <>
          {label.name}
          <br />
        </>
      ))}
    >
      {valueText}
    </TooltipWrapper>
  );
};

interface ISoftwareDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const SoftwareDetailsModal = ({
  details,
  onCancel,
}: ISoftwareDetailsModalProps) => {
  const { labels_include_any, labels_exclude_any } = details;

  return (
    <Modal
      title="Software details"
      width="large"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <DataSet title="Name" value={details.software_title} />
          <DataSet title="Package name" value={details.software_package} />
          <DataSet
            title="Self-Service"
            value={details.self_service ? "Yes" : "No"}
          />
          <DataSet
            title="Target"
            value={
              <TargetValue
                labelIncludeAny={labels_include_any}
                labelExcludeAny={labels_exclude_any}
              />
            }
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default SoftwareDetailsModal;
