import React from "react";

import { IActivityDetails } from "interfaces/activity";
import { ILabelSoftwareTitle } from "interfaces/label";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "software-details-modal";

interface ITargetValueProps {
  labels: ILabelSoftwareTitle[];
}

const TargetValue = ({ labels }: ITargetValueProps) => {
  if (labels.length === 1) {
    return <>labels[0].name</>;
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
      {labels.length} labels
    </TooltipWrapper>
  );
};

const generateTargetTitle = (
  labelIncludeAny?: ILabelSoftwareTitle[],
  labelExcludeAny?: ILabelSoftwareTitle[]
) => {
  if (labelIncludeAny && labelIncludeAny.length > 0) {
    return "Targets (include any)";
  } else if (labelExcludeAny && labelExcludeAny.length > 0) {
    return "Targets (exclude any)";
  }
  return "Targets";
};

const generateTargetValue = (
  labelIncludeAny?: ILabelSoftwareTitle[],
  labelExcludeAny?: ILabelSoftwareTitle[]
) => {
  // handle single label case
  if (labelIncludeAny) {
    return <TargetValue labels={labelIncludeAny} />;
  } else if (labelExcludeAny) {
    return <TargetValue labels={labelExcludeAny} />;
  }
  return "None";
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
  const hasTargets = labels_include_any || labels_exclude_any;

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
          {hasTargets && (
            <DataSet
              title={generateTargetTitle(
                labels_include_any,
                labels_exclude_any
              )}
              value={generateTargetValue(
                labels_include_any,
                labels_exclude_any
              )}
            />
          )}
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
