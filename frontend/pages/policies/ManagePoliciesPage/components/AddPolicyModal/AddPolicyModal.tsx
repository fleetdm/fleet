import React from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import { DEFAULT_POLICIES } from "utilities/constants";

import { IPolicyNew } from "interfaces/policy";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface IAddPolicyModalProps {
  onCancel: () => void;
  router: any;
}

const baseClass = "add-policy-modal";

const AddPolicyModal = ({ onCancel, router }: IAddPolicyModalProps) => {
  const onAddPolicy = (selectedPolicy: IPolicyNew) => {
    const { NEW_POLICY } = PATHS;
    // TODO: Make policy auto populate
    const path = `${NEW_POLICY}?policy=${selectedPolicy}`;
    router.replace(path);
    onCancel();
  };

  const policiesAvailable = DEFAULT_POLICIES.map((policy) => {
    return (
      <Button
        key={policy.key}
        variant="unstyled-modal-query"
        className="modal-policy-button"
        onClick={() => onAddPolicy(policy)}
      >
        <>
          <span className="info__header">{policy.name}</span>
          <span className="info__data">{policy.description}</span>
        </>
      </Button>
    );
  });

  return (
    <Modal
      title="Add a policy"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        Choose a policy template to get started or{" "}
        <Link to={PATHS.NEW_POLICY} className={`${baseClass}__back-link`}>
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

export default AddPolicyModal;
