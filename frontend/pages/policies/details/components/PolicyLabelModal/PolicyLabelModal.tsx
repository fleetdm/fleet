/* eslint-disable @typescript-eslint/no-use-before-define */
import React from "react";
import { Link } from "react-router";

import { ILabelPolicy } from "interfaces/label";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "policy-label-modal";

export interface IPolicyLabelModalProps {
  includeLabels?: ILabelPolicy[];
  includeScopeLabel?: string;
  excludeLabels?: ILabelPolicy[];
  excludeScopeLabel?: string;
  /**
   * When provided, labels render as links to this path; otherwise as plain
   * text.
   * */
  getLabelPath?: (labelId: number) => string;
  onClose: () => void;
}

const PolicyLabelModal = ({
  includeLabels,
  includeScopeLabel,
  excludeLabels,
  excludeScopeLabel,
  getLabelPath,
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
            getLabelPath={getLabelPath}
          />
        )}
        {excludeLabels && excludeScopeLabel && (
          <LabelList
            labels={excludeLabels}
            scopeLabel={excludeScopeLabel}
            description="Policy excludes hosts that"
            getLabelPath={getLabelPath}
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
  getLabelPath?: (labelId: number) => string;
}

const LabelList = ({
  labels,
  scopeLabel,
  description,
  getLabelPath,
}: ILabelListProps): JSX.Element => (
  <div className={`${baseClass}__section`}>
    <span>
      {description} <b>{scopeLabel}</b> of these labels:
    </span>
    <ul className={`${baseClass}__label-list`}>
      {labels.map((label) => (
        <li key={label.id} className={`${baseClass}__label-item`}>
          {getLabelPath ? (
            <Link to={getLabelPath(label.id)}>{label.name}</Link>
          ) : (
            <span>{label.name}</span>
          )}
        </li>
      ))}
    </ul>
  </div>
);

export default PolicyLabelModal;
