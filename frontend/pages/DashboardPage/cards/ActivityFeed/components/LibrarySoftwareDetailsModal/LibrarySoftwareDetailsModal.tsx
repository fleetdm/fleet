import React from "react";

import { IActivityDetails } from "interfaces/activity";
import { ILabelSoftwareTitle } from "interfaces/label";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "library-software-details-modal";

interface ITargetValueProps {
  labelIncludeAny?: ILabelSoftwareTitle[];
  labelExcludeAny?: ILabelSoftwareTitle[];
}

// Shared with global activity VPP details modal
export const TargetTitle = ({
  labelIncludeAny,
  labelExcludeAny,
}: ITargetValueProps) => {
  let suffix = "";

  if (labelIncludeAny) {
    suffix = " (include any)";
  } else if (labelExcludeAny) {
    suffix = " (exclude any)";
  }

  return <>Target{suffix}</>;
};

// Shared with global activity VPP details modal
export const TargetValue = ({
  labelIncludeAny,
  labelExcludeAny,
}: ITargetValueProps) => {
  if (!labelIncludeAny && !labelExcludeAny) {
    return <>All hosts</>;
  }

  let labels: ILabelSoftwareTitle[] = [];
  if (labelIncludeAny) {
    labels = labelIncludeAny;
  } else if (labelExcludeAny) {
    labels = labelExcludeAny;
  }

  // Single label targeted: Show label name
  if (labels.length === 1) {
    return <>{labels[0].name}</>;
  }

  // Multiple labels targeted: Show label count with tooltip of targeted label names
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

interface ILibrarySoftwareDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const LibrarySoftwareDetailsModal = ({
  details,
  onCancel,
}: ILibrarySoftwareDetailsModalProps) => {
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

export default LibrarySoftwareDetailsModal;
