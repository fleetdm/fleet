import React, { useCallback, useRef } from "react";
import { useQuery } from "react-query";
import { omit } from "lodash";

import paths from "router/paths";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";

import { IScript } from "interfaces/script";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";

import { IPaginatedListHandle } from "components/PaginatedList";
import PoliciesPaginatedList, {
  IFormPolicy,
} from "../PoliciesPaginatedList/PoliciesPaginatedList";

const baseClass = "policy-run-script-modal";

interface IScriptDropdownField {
  name: string; // name of the policy
  value: number; // id of the selected script to run with the policy
}

export type IPolicyRunScriptFormData = IFormPolicy[];

interface IPolicyRunScriptModal {
  onExit: () => void;
  onSubmit: (formData: IPolicyRunScriptFormData) => void;
  isUpdating: boolean;
  teamId: number;
  gitOpsModeEnabled?: boolean;
}

const PolicyRunScriptModal = ({
  onExit,
  onSubmit,
  isUpdating,
  teamId,
  gitOpsModeEnabled = false,
}: IPolicyRunScriptModal) => {
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);

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
    if (paginatedListRef.current) {
      onSubmit(paginatedListRef.current.getDirtyItems());
    }
  }, [onSubmit]);

  const onSelectPolicyScript = (
    item: IFormPolicy,
    { value }: IScriptDropdownField
  ) => {
    // Script name needed for error message rendering
    const findScriptNameById = () => {
      const foundScript = availableScripts?.find(
        (script) => script.id === value
      );
      return foundScript ? foundScript.name : "";
    };

    return {
      ...item,
      scriptIdToRun: value,
      scriptNameToRun: findScriptNameById(),
    };
  };

  const availableScriptOptions = availableScripts?.map((script) => ({
    label: script.name,
    value: script.id,
  }));

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
            <CustomLink
              url={getPathWithQueryParams(paths.CONTROLS_SCRIPTS, {
                team_id: teamId,
              })}
              text="Controls &gt; Scripts"
            />{" "}
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
          <div>
            <PoliciesPaginatedList
              ref={paginatedListRef}
              isSelected="runScriptEnabled"
              disableSave={(changedItems) => {
                return changedItems.some(
                  (item) => item.runScriptEnabled && !item.scriptIdToRun
                )
                  ? "Add scripts to all selected policies to save."
                  : false;
              }}
              onToggleItem={(item) => {
                item.runScriptEnabled = !item.runScriptEnabled;
                if (!item.runScriptEnabled) {
                  delete item.scriptIdToRun;
                }
                return item;
              }}
              renderItemRow={(item, onChange) => {
                const formPolicy = {
                  ...item,
                  runScriptEnabled: !!item.scriptIdToRun,
                };
                return item.runScriptEnabled ? (
                  <span
                    onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                      e.stopPropagation();
                    }}
                  >
                    <Dropdown
                      options={availableScriptOptions} // Options filtered for policy's platform(s)
                      value={formPolicy.scriptIdToRun}
                      onChange={({ value }: IScriptDropdownField) =>
                        onChange(
                          onSelectPolicyScript(item, {
                            name: formPolicy.name,
                            value,
                          })
                        )
                      }
                      placeholder="Select script"
                      className={`${baseClass}__script-dropdown`}
                      name={formPolicy.name}
                      parseTarget
                    />
                  </span>
                ) : null;
              }}
              footer={
                <>
                  If{" "}
                  <TooltipWrapper tipContent={compatibleTipContent}>
                    compatible
                  </TooltipWrapper>{" "}
                  with the host, the selected script will run when hosts fail
                  the policy. The script will not run on hosts with scripts
                  disabled, or on hosts with too many pending scripts. Host
                  counts will reset when new scripts are selected.{" "}
                  <CustomLink
                    url="https://fleetdm.com/learn-more-about/policy-automation-run-script"
                    text="Learn more"
                    newTab
                  />
                </>
              }
              isUpdating={isUpdating}
              onSubmit={onUpdate}
              onCancel={onExit}
              teamId={teamId}
            />
          </div>
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
