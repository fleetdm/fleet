import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";

import classnames from "classnames";

import { NotificationContext } from "context/notification";

import { addTeamIdCriteria, IScript } from "interfaces/script";
import { getErrorReason } from "interfaces/errors";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";

import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptBatchSupportedFilters,
  IScriptsResponse,
} from "services/entities/scripts";
import ScriptDetailsModal from "pages/hosts/components/ScriptDetailsModal";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";

import RunScriptBatchPaginatedList from "../RunScriptBatchPaginatedList";
import { IPaginatedListScript } from "../RunScriptBatchPaginatedList/RunScriptBatchPaginatedList";

const baseClass = "run-script-batch-modal";

interface IRunScriptBatchModal {
  runByFilters: boolean; // otherwise, by selectedHostIds
  // since teamId has multiple uses in this component, it's passed in as its own prop and added to
  // `filters` as needed
  filters: Omit<IScriptBatchSupportedFilters, "team_id">;
  teamId: number;
  // If we are on the free tier, we don't want to apply any kind of team filters (since the feature is Premium only).
  isFreeTier?: boolean;
  totalFilteredHostsCount: number;
  selectedHostIds: number[];
  onCancel: () => void;
}

const RunScriptBatchModal = ({
  runByFilters = false,
  filters,
  totalFilteredHostsCount,
  selectedHostIds,
  teamId,
  isFreeTier,
  onCancel,
}: IRunScriptBatchModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const [selectedScript, setSelectedScript] = useState<IScript | undefined>(
    undefined
  );
  const [isUpdating, setIsUpdating] = useState(false);
  const [scriptForDetails, setScriptForDetails] = useState<
    IPaginatedListScript | undefined
  >(undefined);
  // just used to get the total number of scripts, could be optimized by implementing a dedicated scriptsCount endpoint
  const { data: scripts } = useQuery<
    IScriptsResponse,
    Error,
    IScript[],
    IListScriptsQueryKey[]
  >(
    [addTeamIdCriteria({ scope: "scripts" }, teamId, isFreeTier)],
    ({ queryKey }) => {
      return scriptsAPI.getScripts(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      select: (data) => {
        return data.scripts || [];
      },
    }
  );

  const onRunScriptBatch = useCallback(
    async (script: IScript) => {
      setIsUpdating(true);
      const body = runByFilters
        ? // satisfy IScriptBatchSupportedFilters
          {
            script_id: script.id,
            filters: addTeamIdCriteria(filters, teamId, isFreeTier),
          }
        : { script_id: script.id, host_ids: selectedHostIds };
      try {
        await scriptsAPI.runScriptBatch(body);
        renderFlash(
          "success",
          `Script is running on ${
            runByFilters
              ? totalFilteredHostsCount.toLocaleString()
              : selectedHostIds.length.toLocaleString()
          } hosts, or will run as each host comes online. See host details for individual results.`
        );
      } catch (error) {
        let errorMessage = "Could not run script.";
        if (getErrorReason(error).includes("too many hosts")) {
          errorMessage =
            "Could not run script: too many hosts targeted. Please try again with fewer hosts.";
        }
        renderFlash("error", errorMessage);
        // can determine more specific error case with additional call to upcoming summary endpoint
      } finally {
        setIsUpdating(false);
      }
    },
    [renderFlash, selectedHostIds]
  );

  const renderModalContent = () => {
    if (scripts === undefined) {
      return <Spinner />;
    }
    if (!scripts.length) {
      return (
        <EmptyTable
          header="No scripts available for this team"
          info={
            <>
              You can add saved scripts{" "}
              <a
                href={
                  isFreeTier
                    ? "/controls/scripts"
                    : `/controls/scripts?team_id=${teamId}`
                }
              >
                here
              </a>
              .
            </>
          }
        />
      );
    }
    if (!selectedScript) {
      const targetCount = runByFilters
        ? totalFilteredHostsCount
        : selectedHostIds.length;
      return (
        <>
          <p>
            Will run on{" "}
            <b>
              {targetCount.toLocaleString()} host{targetCount > 1 ? "s" : ""}
            </b>
            . You can see individual script results on the host details page.
          </p>
          <RunScriptBatchPaginatedList
            onRunScript={(script) => setSelectedScript(script)}
            isUpdating={isUpdating}
            teamId={teamId}
            isFreeTier={isFreeTier}
            scriptCount={scripts.length}
            setScriptForDetails={setScriptForDetails}
          />
        </>
      );
    }
    const platforms =
      selectedScript.name.indexOf(".ps1") > 0 ? "windows" : "macOS/linux";
    return (
      <div className={`${baseClass}__script-details`}>
        <p>
          <b>{selectedScript.name}</b> will run on compatible hosts ({platforms}
          ).
        </p>
        <div className={`${baseClass}__script-details-actions`}>
          <Button
            onClick={() => {
              onRunScriptBatch(selectedScript);
            }}
            isLoading={isUpdating}
          >
            Run
          </Button>
          <Button
            variant="inverse"
            onClick={() => setSelectedScript(undefined)}
          >
            Cancel
          </Button>
        </div>
      </div>
    );
  };

  const classes = classnames(baseClass, {
    [`${baseClass}__hide-main`]: !!scriptForDetails,
  });

  return (
    <>
      <Modal
        title="Run script"
        onExit={onCancel}
        onEnter={onCancel}
        className={classes}
        disableClosingModal={isUpdating}
      >
        <>
          {renderModalContent()}
          <div className="modal-cta-wrap">
            <Button disabled={isUpdating} onClick={onCancel}>
              Done
            </Button>
          </div>
        </>
      </Modal>
      {!!scriptForDetails && (
        <ScriptDetailsModal
          onCancel={() => setScriptForDetails(undefined)}
          selectedScriptDetails={scriptForDetails}
          suppressSecondaryActions
          customPrimaryButtons={
            <div className="modal-cta-wrap">
              <Button
                onClick={() => {
                  onRunScriptBatch(scriptForDetails);
                }}
                isLoading={isUpdating}
              >
                Run
              </Button>
              <Button
                onClick={() => setScriptForDetails(undefined)}
                variant="inverse"
              >
                Go back
              </Button>
            </div>
          }
        />
      )}
    </>
  );
};

export default RunScriptBatchModal;
