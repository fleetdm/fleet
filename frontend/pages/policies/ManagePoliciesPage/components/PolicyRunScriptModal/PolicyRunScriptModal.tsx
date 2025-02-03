import React, { useCallback, useState } from "react";
import { useQuery } from "react-query";
import { omit } from "lodash";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";

import { IPolicyStats } from "interfaces/policy";
import { IScript } from "interfaces/script";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "policy-run-script-modal";

interface IScriptDropdownField {
  name: string; // name of the policy
  value: number; // id of the selected script to run with the policy
}

interface IFormPolicy {
  name: string;
  id: number;
  runScriptEnabled: boolean;
  scriptIdToRun?: number;
}

export type IPolicyRunScriptFormData = IFormPolicy[];

interface IPolicyRunScriptModal {
  onExit: () => void;
  onSubmit: (formData: IPolicyRunScriptFormData) => void;
  isUpdating: boolean;
  policies: IPolicyStats[];
  teamId: number;
}

const PolicyRunScriptModal = ({
  onExit,
  onSubmit,
  isUpdating,
  policies,
  teamId,
}: IPolicyRunScriptModal) => {
  const [formData, setFormData] = useState<IPolicyRunScriptFormData>(
    policies.map((policy) => ({
      name: policy.name,
      id: policy.id,
      runScriptEnabled: !!policy.run_script,
      scriptIdToRun: policy.run_script?.id,
    }))
  );

  const anyEnabledWithoutSelection = formData.some(
    (policy) => policy.runScriptEnabled && !policy.scriptIdToRun
  );

  const {
    data: availableScripts,
    isLoading: isLoadingAvailableScripts,
    isError: isAvailableScriptsError,
  } = useQuery<IScriptsResponse, Error, IScript[], [IListScriptsQueryKey]>(
    [
      {
        scope: "scripts",
        team_id: teamId,
      },
    ],
    ({ queryKey: [queryKey] }) =>
      scriptsAPI.getScripts(omit(queryKey, "scope")),
    {
      select: (data) => data.scripts,
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const onUpdate = useCallback(() => {
    onSubmit(formData);
  }, [formData, onSubmit]);

  const onChangeEnableRunScript = useCallback(
    (newVal: { policyId: number; value: boolean }) => {
      const { policyId, value } = newVal;
      setFormData(
        formData.map((policy) => {
          if (policy.id === policyId) {
            return {
              ...policy,
              runScriptEnabled: value,
              scriptIdToRun: value ? policy.scriptIdToRun : undefined,
            };
          }
          return policy;
        })
      );
    },
    [formData]
  );

  const onSelectPolicyScript = useCallback(
    ({ name, value }: IScriptDropdownField) => {
      const [policyName, scriptId] = [name, value];
      setFormData(
        formData.map((policy) => {
          if (policy.name === policyName) {
            return { ...policy, scriptIdToRun: scriptId };
          }
          return policy;
        })
      );
    },
    [formData]
  );

  const availableScriptOptions = availableScripts?.map((script) => ({
    label: script.name,
    value: script.id,
  }));

  const renderPolicyRunScriptOption = (policy: IFormPolicy) => {
    const {
      name: policyName,
      id: policyId,
      runScriptEnabled: enabled,
      scriptIdToRun,
    } = policy;

    return (
      <li
        className={`${baseClass}__policy-row policy-row`}
        id={`policy-row--${policyId}`}
        key={`${policyId}-${enabled}`}
      >
        <Checkbox
          value={enabled}
          name={policyName}
          onChange={() => {
            onChangeEnableRunScript({
              policyId,
              value: !enabled,
            });
          }}
        >
          <TooltipTruncatedText value={policyName} />
        </Checkbox>
        {enabled && (
          <Dropdown
            options={availableScriptOptions}
            value={scriptIdToRun}
            onChange={onSelectPolicyScript}
            placeholder="Select script"
            className={`${baseClass}__script-dropdown`}
            name={policyName}
            parseTarget
          />
        )}
      </li>
    );
  };

  const renderContent = () => {
    if (isAvailableScriptsError) {
      return <DataError />;
    }
    if (isLoadingAvailableScripts) {
      return <Spinner />;
    }
    if (!availableScripts?.length) {
      return (
        <div className={`${baseClass}__no-scripts`}>
          <b>No scripts available for install</b>
          <div>
            Go to{" "}
            <a href={`/controls/scripts?team_id=${teamId}`}>
              Controls &gt; Scripts
            </a>{" "}
            to add scripts to this team.
          </div>
        </div>
      );
    }

    const compatibleTipContent = (
      <>
        Shell (.sh) for macOS and Linux.
        <br />
        PowerShell (.ps1) for Windows.
      </>
    );

    return (
      <div className={`${baseClass} form`}>
        <div className="form-field">
          <div className="form-field__label">Policies:</div>
          <ul className="automated-policies-section">
            {formData.map((policyData) =>
              renderPolicyRunScriptOption(policyData)
            )}
          </ul>
          <span className="form-field__help-text">
            If{" "}
            <TooltipWrapper tipContent={compatibleTipContent}>
              compatible
            </TooltipWrapper>{" "}
            with the host, the selected script will run when hosts fail the
            policy. The script will not run on hosts with scripts disabled, or
            on hosts with too many pending scripts. Host counts will reset when
            new scripts are selected.{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/policy-automation-run-script"
              text="Learn more"
              newTab
            />
          </span>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            onClick={onUpdate}
            className="save-loading"
            isLoading={isUpdating}
            disabled={anyEnabledWithoutSelection}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    );
  };
  return (
    <Modal
      title="Run script"
      className={baseClass}
      onExit={onExit}
      onEnter={onUpdate}
      width="large"
      isContentDisabled={isUpdating}
    >
      {renderContent()}
    </Modal>
  );
};

export default PolicyRunScriptModal;
