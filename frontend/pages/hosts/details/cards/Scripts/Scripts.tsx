import React, { useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import scriptsAPI, {
  IHostScript,
  IHostScriptsResponse,
} from "services/entities/scripts";
import { IApiError, IError } from "interfaces/errors";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import DataError from "components/DataError";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import ScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptDetailsModal";

import { generateDataSet, generateTableHeaders } from "./ScriptsTableConfig";

const baseClass = "host-scripts-section";

interface IScriptsProps {
  hostId?: number;
  page?: number;
  isHostOnline: boolean;
  router: InjectedRouter;
}

const Scripts = ({ hostId, page = 0, isHostOnline, router }: IScriptsProps) => {
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  // used to track the current script execution id we want to show in the show
  // details modal.
  const scriptExecutionId = useRef<string | null>(null);

  const { renderFlash } = useContext(NotificationContext);

  const {
    data: hostScriptResponse,
    isLoading: isLoadingScriptData,
    isError: isErrorScriptData,
    refetch: refetchScriptsData,
  } = useQuery<IHostScriptsResponse, IError>(
    ["scripts", hostId, page],
    () => scriptsAPI.getHostScripts(hostId as number, page),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: Boolean(hostId),
    }
  );

  if (!hostId) return null;

  const onQueryChange = (data: ITableQueryData) => {
    router.push(`${PATHS.HOST_SCRIPTS(hostId)}?page=${data.pageIndex}`);
  };

  const onActionSelection = async (action: string, script: IHostScript) => {
    switch (action) {
      case "showDetails":
        if (!script.last_execution) return;
        scriptExecutionId.current = script.last_execution.execution_id;
        setShowScriptDetailsModal(true);
        break;
      case "run":
        try {
          await scriptsAPI.runScript(script.script_id);
          refetchScriptsData();
        } catch (e) {
          const error = e as AxiosResponse<IApiError>;
          renderFlash("error", error.data.errors[0].reason);
        }
        break;
      default:
        break;
    }
  };

  const onCancelScriptDetailsModal = () => {
    setShowScriptDetailsModal(false);
    scriptExecutionId.current = null;
  };

  if (isErrorScriptData) {
    return <DataError card />;
  }

  const scriptHeaders = generateTableHeaders(onActionSelection);
  const data = generateDataSet(hostScriptResponse?.scripts || [], isHostOnline);

  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      <h2>Scripts</h2>
      {data && data.length === 0 ? (
        <EmptyTable
          header="No scripts are available for this host"
          info="Expecting to see scripts? Try selecting “Refetch” to ask this host to report new vitals."
        />
      ) : (
        <TableContainer
          resultsTitle=""
          emptyComponent={() => <></>}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          columns={scriptHeaders}
          data={data}
          isLoading={isLoadingScriptData}
          onQueryChange={onQueryChange}
          disableNextPage={hostScriptResponse?.meta.has_next_results}
          defaultPageIndex={page}
          disableCount
        />
      )}

      {showScriptDetailsModal && scriptExecutionId.current && (
        <ScriptDetailsModal
          scriptExecutionId={scriptExecutionId.current}
          onCancel={onCancelScriptDetailsModal}
        />
      )}
    </Card>
  );
};

export default Scripts;
