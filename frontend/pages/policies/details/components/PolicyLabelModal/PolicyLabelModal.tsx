import React from "react";

import { ILabelPolicy } from "interfaces/label";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "policy-label-modal";

interface IPolicyLabelModalProps {
  includeLabels?: ILabelPolicy[];
  includeScopeLabel?: string;
  excludeLabels?: ILabelPolicy[];
  excludeScopeLabel?: string;
  /** When provided, labels render as clickable links; otherwise as plain text. */
  onLabelClick?: (labelId: number) => void;
  onClose: () => void;
}

const renderLabelList = (
  labels: ILabelPolicy[],
  scopeLabel: string,
  description: string,
  onLabelClick?: (labelId: number) => void
) => (
  <div className={`${baseClass}__section`}>
    <span className={`${baseClass}__scope-description`}>
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
      <div className={baseClass}>
        {includeLabels &&
          includeScopeLabel &&
          renderLabelList(
            includeLabels,
            includeScopeLabel,
            "Policy targets hosts that",
            onLabelClick
          )}
        {excludeLabels &&
          excludeScopeLabel &&
          renderLabelList(
            excludeLabels,
            excludeScopeLabel,
            "Policy excludes hosts that",
            onLabelClick
          )}
        <div className="modal-cta-wrap">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyLabelModal;
