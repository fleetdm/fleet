import React, { useCallback, useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { getErrorReason, IApiError } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { IHostScript } from "interfaces/script";
import { IUser } from "interfaces/user";

import scriptsAPI, {
  IHostScriptsQueryKey,
  IHostScriptsResponse,
} from "services/entities/scripts";

import Button from "components/buttons/Button";
import DataError from "components/DataError/DataError";
import EmptyTable from "components/EmptyTable";
import Modal from "components/Modal";
import Spinner from "components/Spinner/Spinner";

import TableContainer, {
  ITableQueryData,
} from "components/TableContainer/TableContainer";

import { generateTableColumnConfigs } from "./ScriptsTableConfig";

const baseClass = "run-script-modal";

interface IScriptsProps {
  currentUser: IUser | null;
  host: IHost;
  scriptDetailsId: string;
  setScriptDetailsId: React.Dispatch<React.SetStateAction<string>>;
  onClose: () => void;
}

const EmptyComponent = () => <></>;

const RunScriptModal = ({
  currentUser,
  host,
  scriptDetailsId,
  setScriptDetailsId,
  onClose,
}: IScriptsProps) => {
  const [page, setPage] = useState<number>(0);
  const [runScriptRequested, setRunScriptRequested] = useState(false);

  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const {
    data: hostScriptResponse,
    isError,
    isLoading,
    isFetching,
    refetch: refetchHostScripts,
  } = useQuery<
    IHostScriptsResponse,
    IApiError,
    IHostScriptsResponse,
    IHostScriptsQueryKey[]
  >(
    [{ scope: "host_scripts", host_id: host.id, page, per_page: 10 }],
    ({ queryKey }) => scriptsAPI.getHostScripts(queryKey[0]),
    {
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: 3000,
      onSuccess: () => {
        setRunScriptRequested(false);
      },
    }
  );

  const onSelectAction = useCallback(
    async (action: string, script: IHostScript) => {
      switch (action) {
        case "showDetails": {
          setScriptDetailsId(script.last_execution?.execution_id || "");
          break;
        }
        case "run": {
          try {
            setRunScriptRequested(true);
            await scriptsAPI.runScript({
              host_id: host.id,
              script_id: script.script_id,
            });
            renderFlash(
              "success",
              "Script is running or will run when the host comes online."
            );
            refetchHostScripts();
          } catch (e) {
            renderFlash("error", getErrorReason(e));
            setRunScriptRequested(false);
          }
          break;
        }
        default: // do nothing
      }
    },
    [host.id, refetchHostScripts, renderFlash, setScriptDetailsId]
  );

  const onQueryChange = useCallback(({ pageIndex }: ITableQueryData) => {
    setPage(pageIndex);
  }, []);

  const scriptColumnConfigs = useMemo(
    () =>
      generateTableColumnConfigs(
        currentUser,
        host.team_id,
        !!config?.server_settings?.scripts_disabled,
        onSelectAction
      ),
    [currentUser, host.team_id, config, onSelectAction]
  );

  if (!config) return null;

  const isShowingScriptDetails = !!scriptDetailsId; // used to set css visibility for this modal to hidden when the script details modal is open
  const tableData = hostScriptResponse?.scripts;

  return (
    <Modal
      title="Run script"
      onExit={onClose}
      onEnter={onClose}
      className={`${baseClass}`}
      isHidden={isShowingScriptDetails}
      isLoading={runScriptRequested || isFetching || isLoading}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          {isLoading && <Spinner />}
          {!isLoading && isError && <DataError />}
          {!isLoading && !isError && (!tableData || tableData.length === 0) && (
            <EmptyTable
              header="No scripts are available for this host"
              info="Expecting to see scripts? Try selecting “Refetch” to ask the host to report new vitals."
            />
          )}
          {!isLoading && !isError && tableData && tableData.length > 0 && (
            <TableContainer
              resultsTitle=""
              emptyComponent={EmptyComponent}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              columnConfigs={scriptColumnConfigs}
              data={tableData}
              isLoading={runScriptRequested || isFetching}
              onQueryChange={onQueryChange}
              disableNextPage={!hostScriptResponse?.meta.has_next_results}
              defaultPageIndex={page}
              pageSize={10}
              disableCount
              disableTableHeader
            />
          )}
        </div>
        <div className={`modal-cta-wrap`}>
          <Button onClick={onClose} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default React.memo(RunScriptModal);
