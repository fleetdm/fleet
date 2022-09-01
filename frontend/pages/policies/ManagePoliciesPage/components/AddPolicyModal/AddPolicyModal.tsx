import React, { useCallback, useContext } from "react";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router/lib/Router";

import { DEFAULT_POLICY, DEFAULT_POLICIES } from "utilities/constants";

import { IPolicyNew } from "interfaces/policy";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface IAddPolicyModalProps {
  onCancel: () => void;
  router: InjectedRouter; // v3
  teamId: number;
  teamName?: string;
}

const baseClass = "add-policy-modal";

const AddPolicyModal = ({
  onCancel,
  router,
  teamId,
  teamName,
}: IAddPolicyModalProps): JSX.Element => {
  const { currentTeam } = useContext(AppContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryPlatform,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  const onAddPolicy = (selectedPolicy: IPolicyNew) => {
    teamName
      ? setLastEditedQueryName(`${selectedPolicy.name} (${teamName})`)
      : setLastEditedQueryName(selectedPolicy.name);
    setLastEditedQueryDescription(selectedPolicy.description);
    setLastEditedQueryBody(selectedPolicy.query);
    setLastEditedQueryResolution(selectedPolicy.resolution);
    setPolicyTeamId(teamId);
    setLastEditedQueryPlatform(selectedPolicy.platform || null);
    router.push(PATHS.NEW_POLICY);
  };

  const onCreateYourOwnPolicyClick = useCallback(() => {
    setPolicyTeamId(currentTeam?.id || 0);
    setLastEditedQueryBody(DEFAULT_POLICY.query);
    router.push(PATHS.NEW_POLICY);
  }, [currentTeam]);

  const policiesAvailable = DEFAULT_POLICIES.map((policy: IPolicyNew) => {
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
    <Modal title="Add a policy" onExit={onCancel} className={baseClass}>
      <>
        Choose a policy template to get started or{" "}
        <Button variant="text-link" onClick={onCreateYourOwnPolicyClick}>
          create your own policy
        </Button>
        .
        <div className={`${baseClass}__policy-selection`}>
          {policiesAvailable}
        </div>
      </>
    </Modal>
  );
};

export default AddPolicyModal;
