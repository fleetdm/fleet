import React, { useCallback, useContext, useState } from "react";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router/lib/Router";

import { DEFAULT_POLICY, DEFAULT_POLICIES } from "pages/policies/constants";

import { IPolicyNew } from "interfaces/policy";
import { SelectedPlatform } from "interfaces/platform";

import { PolicyContext } from "context/policy";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import CustomLink from "components/CustomLink";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";

export interface IAddPolicyModalProps {
  onCancel: () => void;
  router: InjectedRouter; // v3
  // API context, all teams: undefined
  teamId: number | undefined;
  teamName?: string;
}

const CONTRIBUTE_TO_POLICIES_DOCS_URL =
  "https://www.fleetdm.com/contribute-to/policies";

const PLATFORM_FILTER_OPTIONS = [
  {
    label: "All platforms",
    value: "all",
  },
  {
    label: "macOS",
    value: "darwin",
  },
  {
    label: "Windows",
    value: "windows",
  },
  {
    label: "Linux",
    value: "linux",
  },
  {
    label: "ChromeOS",
    value: "chrome",
  },
];

const baseClass = "add-policy-modal";

const AddPolicyModal = ({
  onCancel,
  router,
  teamId,
  teamName,
}: IAddPolicyModalProps): JSX.Element => {
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
    setLastEditedQueryId,
    setPolicyTeamId,
    setDefaultPolicy,
  } = useContext(PolicyContext);

  const [filteredPolicies, setFilteredPolicies] = useState(DEFAULT_POLICIES);
  const [platform, setPlatform] = useState<SelectedPlatform>("all");

  const onAddPolicy = (selectedPolicy: IPolicyNew) => {
    setDefaultPolicy(true);
    teamName
      ? setLastEditedQueryName(`${selectedPolicy.name} (${teamName})`)
      : setLastEditedQueryName(selectedPolicy.name);
    setLastEditedQueryDescription(selectedPolicy.description);
    setLastEditedQueryBody(selectedPolicy.query);
    setLastEditedQueryResolution(selectedPolicy.resolution);
    setLastEditedQueryCritical(selectedPolicy.critical || false);
    setLastEditedQueryId(null);
    setPolicyTeamId(
      teamId === API_ALL_TEAMS_ID ? APP_CONTEXT_ALL_TEAMS_ID : teamId
    );
    setLastEditedQueryPlatform(selectedPolicy.platform || null);
    router.push(
      teamId === API_ALL_TEAMS_ID
        ? PATHS.NEW_POLICY
        : `${PATHS.NEW_POLICY}?team_id=${teamId}`
    );
  };

  const onCreateYourOwnPolicyClick = useCallback(() => {
    setPolicyTeamId(
      teamId === API_ALL_TEAMS_ID ? APP_CONTEXT_ALL_TEAMS_ID : teamId
    );
    setLastEditedQueryBody(DEFAULT_POLICY.query);
    setLastEditedQueryId(null);
    router.push(
      teamId === API_ALL_TEAMS_ID
        ? PATHS.NEW_POLICY
        : `${PATHS.NEW_POLICY}?team_id=${teamId}`
    );
  }, [
    router,
    setLastEditedQueryBody,
    setLastEditedQueryId,
    setPolicyTeamId,
    teamId,
  ]);

  const onPlatformFilterChange = (platformSelected: SelectedPlatform) => {
    if (platformSelected === "all") {
      setFilteredPolicies(DEFAULT_POLICIES);
    } else {
      // Note: Default policies currently map to a single platform
      const policiesFilteredByPlatform = DEFAULT_POLICIES.filter((policy) => {
        return policy.platform === platformSelected;
      });
      setFilteredPolicies(policiesFilteredByPlatform);
    }
    setPlatform(platformSelected);
  };

  const filteredPoliciesCount = filteredPolicies.length;

  const filteredPoliciesList = filteredPolicies.map((policy: IPolicyNew) => {
    return (
      <Button
        key={policy.key}
        variant="unstyled-modal-query"
        className="modal-policy-button"
        onClick={() => onAddPolicy(policy)}
      >
        <>
          <div className={`${baseClass}__policy-name`}>
            <span className="info__header">{policy.name}</span>
            {policy.mdm_required && (
              <span className={`${baseClass}__mdm-policy`}>Requires MDM</span>
            )}
          </div>
          <span className="info__data">{policy.description}</span>
        </>
      </Button>
    );
  });

  const renderNoResults = () => {
    return (
      <>
        There are no results that match your filters.{" "}
        <CustomLink
          text="Everyone can contribute"
          url={CONTRIBUTE_TO_POLICIES_DOCS_URL}
          newTab
        />
      </>
    );
  };

  return (
    <Modal
      title="Add a policy"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <>
        <div className={`${baseClass}__description`}>
          Choose a policy template to get started or{" "}
          <Button variant="text-link" onClick={onCreateYourOwnPolicyClick}>
            create your own policy
          </Button>
          .
        </div>
        <Dropdown
          value={platform}
          className={`${baseClass}__platform-dropdown`}
          options={PLATFORM_FILTER_OPTIONS}
          searchable={false}
          onChange={onPlatformFilterChange}
          iconName="filter"
        />
        <div className={`${baseClass}__policy-selection`}>
          {filteredPoliciesCount > 0 ? filteredPoliciesList : renderNoResults()}
        </div>
      </>
    </Modal>
  );
};

export default AddPolicyModal;
