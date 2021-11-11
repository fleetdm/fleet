import React from "react";

import { useDispatch } from "react-redux";

import { DEFAULT_POLICIES } from "utilities/constants";

import { IHost } from "interfaces/host";
import { IPolicyNew } from "interfaces/policy";
import { IQuery } from "interfaces/query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface ISelectQueryModalProps {
  host: IHost;
  onCancel: () => void;
  queries: IQuery[] | [];
  queryErrors: any | null;
  isOnlyObserver: boolean | undefined;
}

const baseClass = "add-policy-modal";

const SelectPolicyModal = ({ onCancel }: ISelectQueryModalProps) => {
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
    // add policy
    // close modal
  };

  return (
    <Modal
      title="Add a policy"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        Choose a policy template to get started or create your own policy.
        <div>{policiesAvailable}</div>
      </>
    </Modal>
  );
};

export default SelectPolicyModal;
