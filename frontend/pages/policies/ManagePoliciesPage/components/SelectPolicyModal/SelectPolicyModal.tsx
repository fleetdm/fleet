import React from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import { DEFAULT_POLICIES } from "utilities/constants";

import { IHost } from "interfaces/host";
import { IPolicyNew } from "interfaces/policy";
import { IQuery } from "interfaces/query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface ISelectPolicyModalProps {
  onCancel: () => void;
  router: any;
}

const baseClass = "add-policy-modal";

const SelectPolicyModal = ({ onCancel, router }: ISelectPolicyModalProps) => {
  const policiesAvailable = DEFAULT_POLICIES.map((policy) => {
    return (
      <Button
        key={policy.key}
        variant="unstyled-modal-query"
        className="modal-policy-button"
        onClick={() => onSelectPolicy(policy)}
      >
        <>
          <span className="info__header">{policy.name}</span>
          <span className="info__data">{policy.description}</span>
        </>
      </Button>
    );
  });
  const onSelectPolicy = (selectedPolicy: IPolicyNew) => {
    const { NEW_QUERY } = PATHS;
    // TODO: Change to NEW_POLICY after Martavis PR is merged
    // Make policy auto populate
    const path = `${NEW_QUERY}?policy=${selectedPolicy}`;
    router.replace(path);
    onCancel();
  };

  return (
    <Modal
      title="Add a policy"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        Choose a policy template to get started or{" "}
        {/* TODO: Change to NEW_POLICY after Martavis PR is merged */}
        <Link to={PATHS.NEW_QUERY} className={`${baseClass}__back-link`}>
          create your own policy
        </Link>
        .
        <div className={`${baseClass}__policy-selection`}>
          {policiesAvailable}
        </div>
      </>
    </Modal>
  );
};

export default SelectPolicyModal;
