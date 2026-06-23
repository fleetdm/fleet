/* eslint-disable @typescript-eslint/no-use-before-define */
import React from "react";

import { ILabelPolicy } from "interfaces/label";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "policy-label-modal";

export interface IPolicyLabelModalProps {
  includeLabels?: ILabelPolicy[];
  includeScopeLabel?: string;
  excludeLabels?: ILabelPolicy[];
  excludeScopeLabel?: string;
  /** When provided, labels render as clickable links; otherwise as plain text. */
  onLabelClick?: (labelId: number) => void;
  onClose: () => void;
}

const PolicyLabelModal = ({
  includeLabels,
  includeScopeLabel,
  excludeLabels,
  excludeScopeLabel,
  onLabelClick,
  onClose,
}: IPolicyLabelModalProps): JSX.Element => {
  return (
    <Modal
      title="Labels"
      onExit={onClose}
      onEnter={onClose}
      className={baseClass}
    >
      <div className={`${baseClass}__body`}>
        {includeLabels && includeScopeLabel && (
          <LabelList
            labels={includeLabels}
            scopeLabel={includeScopeLabel}
            description="Policy targets hosts that"
            onLabelClick={onLabelClick}
          />
        )}
        {excludeLabels && excludeScopeLabel && (
          <LabelList
            labels={excludeLabels}
            scopeLabel={excludeScopeLabel}
            description="Policy excludes hosts that"
            onLabelClick={onLabelClick}
          />
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

interface ILabelListProps {
  labels: ILabelPolicy[];
  scopeLabel: string;
  description: string;
  onLabelClick?: (labelId: number) => void;
}

const LabelList = ({
  labels,
  scopeLabel,
  description,
  onLabelClick,
}: ILabelListProps): JSX.Element => (
  <div className={`${baseClass}__section`}>
    <span>
      {description} <b>{scopeLabel}</b> of these labels:
    </span>
    <ul className={`${baseClass}__label-list`}>
      {labels.map((label) => (
        <li key={label.id} className={`${baseClass}__label-item`}>
          {onLabelClick ? (
            <Button variant="link" onClick={() => onLabelClick(label.id)}>
              {label.name}
            </Button>
          ) : (
            <span>{label.name}</span>
          )}
        </li>
      ))}
    </ul>
  </div>
);

export default PolicyLabelModal;
