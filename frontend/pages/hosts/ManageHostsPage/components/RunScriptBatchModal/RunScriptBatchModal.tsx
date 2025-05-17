import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";

import classnames from "classnames";

import { NotificationContext } from "context/notification";

import { IScript } from "interfaces/script";

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
  onCancel,
}: IRunScriptBatchModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUpdating, setIsUpdating] = useState(false);
  const [scriptForDetails, setScriptForDetails] = useState<
    IPaginatedListScript | undefined
  >(undefined);
  // just used to get total number of scripts, could be optimized by implementing a dedicated scriptsCount endpoint
  const { data: scripts } = useQuery<
    IScriptsResponse,
    Error,
    IScript[],
    IListScriptsQueryKey[]
  >(
    [
      {
        scope: "scripts",
        team_id: teamId,
      },
    ],
    ({ queryKey }) => {
      return scriptsAPI.getScripts(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      select: (data) => data.scripts || [],
    }
  );

  const onRunScriptBatch = useCallback(
    async (script: IScript) => {
      setIsUpdating(true);
      const body = runByFilters
        ? // satisfy IScriptBatchSupportedFilters
          { script_id: script.id, filters: { ...filters, team_id: teamId } }
        : { script_id: script.id, host_ids: selectedHostIds };
      try {
        await scriptsAPI.runScriptBatch(body);
        renderFlash(
          "success",
          `Script is running on ${
            runByFilters ? totalFilteredHostsCount : selectedHostIds.length
          } hosts, or will run as each host comes online. See host details for individual results.`
        );
      } catch (error) {
        renderFlash("error", "Could not run script.");
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
              <a href={`/controls/scripts?team_id=${teamId}`}>here</a>.
            </>
          }
        />
      );
    }
    const targetCount = runByFilters
      ? totalFilteredHostsCount
      : selectedHostIds.length;
    return (
      <>
        <p>
          Will run on{" "}
          <b>
            {targetCount} host{targetCount > 1 ? "s" : ""}
          </b>
          . You can see individual script results on the host details page.
        </p>
        <RunScriptBatchPaginatedList
          onRunScript={onRunScriptBatch}
          isUpdating={isUpdating}
          teamId={teamId}
          scriptCount={scripts.length}
          setScriptForDetails={setScriptForDetails}
        />
      </>
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
