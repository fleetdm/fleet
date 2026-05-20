import React, { useCallback, useContext, useMemo } from "react";

import PATHS from "router/paths";

import { AppContext } from "context/app";

import { IHostScript } from "interfaces/script";
import { IUser } from "interfaces/user";
import { IHostScriptsResponse } from "services/entities/scripts";

import permissions from "utilities/permissions";
import { getPathWithQueryParams } from "utilities/url";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError/DataError";
import EmptyState from "components/EmptyState";
import Modal from "components/Modal";
import Spinner from "components/Spinner/Spinner";

import TableContainer, {
  ITableQueryData,
} from "components/TableContainer/TableContainer";

import { generateTableColumnConfigs } from "./ScriptsTableConfig";

const baseClass = "run-script-modal";

interface IRunScriptModalProps {
  currentUser: IUser | null;
  hostTeamId: number | null;
  onClose: () => void;
  page: number;
  setPage: React.Dispatch<React.SetStateAction<number>>;
  hostScriptResponse?: IHostScriptsResponse;
  isFetchingHostScripts: boolean;
  isLoadingHostScripts: boolean;
  isError: boolean;
  onClickViewScript: (scriptDetails: IHostScript) => void;
  onClickRunDetails: (scriptExecutionId: string) => void;
  onClickRun: (script: IHostScript) => void;
  isRunningScript: boolean;
  isHidden: boolean;
}

const EmptyComponent = () => <></>;

const RunScriptModal = ({
  currentUser,
  hostTeamId,
  onClose,
  page,
  setPage,
  hostScriptResponse,
  isFetchingHostScripts,
  isLoadingHostScripts,
  isError,
  onClickViewScript,
  onClickRunDetails,
  onClickRun,
  isRunningScript,
  isHidden = false,
}: IRunScriptModalProps) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const onSelectAction = useCallback(
    async (action: string, script: IHostScript) => {
      switch (action) {
        case "showRunDetails": {
          script.last_execution?.execution_id &&
            onClickRunDetails(script.last_execution?.execution_id);
          break;
        }
        case "run": {
          onClickRun(script);
          break;
        }
        default: // do nothing
      }
    },
    [onClickRun, onClickRunDetails]
  );

  const onQueryChange = useCallback(({ pageIndex }: ITableQueryData) => {
    setPage(pageIndex);
  }, []);

  const scriptColumnConfigs = useMemo(
    () =>
      generateTableColumnConfigs(
        currentUser,
        hostTeamId,
        // 4.81+ users won't reach this modal if scripts are disabled
        // Intentionally left disabled actions in as a safeguard
        !!config?.server_settings?.scripts_disabled,
        onClickViewScript,
        onSelectAction
      ),
    [
      currentUser,
      hostTeamId,
      config?.server_settings?.scripts_disabled,
      onClickViewScript,
      onSelectAction,
    ]
  );

  if (!config) return null;

  const tableData = hostScriptResponse?.scripts;

  // Technicians can run scripts but can't upload them — hide the "Add a script"
  // link for them since it would lead to a page they can't act on.
  const isTechnician =
    !!currentUser &&
    (permissions.isGlobalTechnician(currentUser) ||
      permissions.isTeamTechnician(currentUser, hostTeamId));

  return (
    <Modal
      title="Run script"
      onExit={onClose}
      onEnter={onClose}
      className={`${baseClass}`}
      isLoading={isFetchingHostScripts || isLoadingHostScripts}
      isHidden={isHidden}
    >
      <div className={`${baseClass}__modal-content`}>
        {isLoadingHostScripts && <Spinner />}
        {!isLoadingHostScripts && isError && <DataError />}
        {!isLoadingHostScripts &&
          !isError &&
          (!tableData || tableData.length === 0) && (
            <EmptyState
              variant="header-list"
              header="No scripts available"
              info={
                isTechnician ? (
                  "Ask your admin to add a script for this host."
                ) : (
                  <>
                    <CustomLink
                      url={getPathWithQueryParams(
                        PATHS.CONTROLS_SCRIPTS,
                        isPremiumTier ? { fleet_id: hostTeamId } : undefined
                      )}
                      text="Add a script"
                    />{" "}
                    available to this host.
                  </>
                )
              }
            />
          )}
        {!isLoadingHostScripts &&
          !isError &&
          tableData &&
          tableData.length > 0 && (
            <TableContainer
              resultsTitle=""
              emptyComponent={EmptyComponent}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              columnConfigs={scriptColumnConfigs}
              data={tableData}
              isLoading={isRunningScript || isFetchingHostScripts}
              onQueryChange={onQueryChange}
              disableNextPage={!hostScriptResponse?.meta.has_next_results}
              pageIndex={page}
              pageSize={10}
              disableCount
              disableTableHeader
            />
          )}
      </div>
      <div className="modal-cta-wrap">
        <Button onClick={onClose}>Close</Button>
      </div>
    </Modal>
  );
};

export default React.memo(RunScriptModal);
