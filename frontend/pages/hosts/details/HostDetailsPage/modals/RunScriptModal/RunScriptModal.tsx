import React, { useCallback, useContext, useMemo } from "react";
import { browserHistory } from "react-router";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError/DataError";
import EmptyState from "components/EmptyState";
import Modal from "components/Modal";
import Spinner from "components/Spinner/Spinner";
import TableContainer, {
  ITableQueryData,
} from "components/TableContainer/TableContainer";
import { AppContext } from "context/app";
import { isLinuxLike } from "interfaces/platform";
import { IHostScript } from "interfaces/script";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { IUser } from "interfaces/user";
import PATHS from "router/paths";
import { IHostScriptsResponse } from "services/entities/scripts";
import permissions from "utilities/permissions";
import { getPathWithQueryParams } from "utilities/url";

import { generateTableColumnConfigs } from "./ScriptsTableConfig";

const baseClass = "run-script-modal";

interface IRunScriptModalProps {
  currentUser: IUser | null;
  hostTeamId: number | null;
  hostPlatform?: string;
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

// Mirrors the extension filter applied server-side when listing a host's scripts.
const getCompatibleScriptTypes = (platform?: string) => {
  if (!platform) return undefined;
  if (platform === "windows") return "PowerShell (.ps1)";
  if (platform === "darwin" || isLinuxLike(platform)) {
    return "shell (.sh) and Python (.py)";
  }
  return undefined;
};

const RunScriptModal = ({
  currentUser,
  hostTeamId,
  hostPlatform,
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

  // Only admins and maintainers (global or on the host's team) can upload scripts,
  // so the "Add a script" link is hidden for everyone else (e.g. technicians).
  const canAddScript =
    !!currentUser &&
    (permissions.isGlobalAdmin(currentUser) ||
      permissions.isGlobalMaintainer(currentUser) ||
      permissions.isTeamAdmin(currentUser, hostTeamId) ||
      permissions.isTeamMaintainer(currentUser, hostTeamId));

  const addScriptUrl = getPathWithQueryParams(
    PATHS.CONTROLS_SCRIPTS,
    isPremiumTier
      ? { fleet_id: hostTeamId ?? APP_CONTEXT_NO_TEAM_ID }
      : undefined
  );

  const compatibleScriptTypes = getCompatibleScriptTypes(hostPlatform);

  const renderEmptyStateInfo = () => {
    const addScriptInfo = canAddScript ? (
      <>
        <CustomLink url={addScriptUrl} text="Add a script" />.
      </>
    ) : (
      "Ask your admin to add a script for this host."
    );
    if (!compatibleScriptTypes) return addScriptInfo;
    return (
      <>
        This host can only run {compatibleScriptTypes} scripts. {addScriptInfo}
      </>
    );
  };

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
              header={
                compatibleScriptTypes
                  ? "No compatible scripts"
                  : "No scripts available"
              }
              info={renderEmptyStateInfo()}
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
        {canAddScript && !!tableData?.length && (
          <Button
            variant="secondary"
            icon="plus"
            onClick={() => browserHistory.push(addScriptUrl)}
          >
            Add script
          </Button>
        )}
      </div>
    </Modal>
  );
};

export default React.memo(RunScriptModal);
