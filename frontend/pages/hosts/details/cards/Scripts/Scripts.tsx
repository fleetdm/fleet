import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import scriptsAPI, {
  IHostScript,
  IHostScriptsResponse,
} from "services/entities/scripts";
import { IApiError } from "interfaces/errors";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import DataError from "components/DataError";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import { IHost } from "interfaces/host";
import { IUser } from "interfaces/user";

import {
  generateDataSet,
  generateTableColumnConfigs,
} from "./ScriptsTableConfig";

const baseClass = "host-scripts-section";

interface IScriptsProps {
  currentUser: IUser | null;
  host?: IHost;
  router: InjectedRouter;
  page?: number;
  onShowDetails: (scriptExecutionId: string) => void;
}

const Scripts = ({
  currentUser,
  host,
  page = 0,
  router,
  onShowDetails,
}: IScriptsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const hostId = host?.id;

  const {
    data: hostScriptResponse,
    isLoading: isLoadingScriptData,
    isError: isErrorScriptData,
    refetch: refetchScriptsData,
  } = useQuery<IHostScriptsResponse, IApiError>(
    ["scripts", hostId, page],
    () => scriptsAPI.getHostScripts(hostId as number, page),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: Boolean(hostId),
    }
  );

  if (!host) return null;

  const onQueryChange = (data: ITableQueryData) => {
    router.push(`${PATHS.HOST_SCRIPTS(host.id)}?page=${data.pageIndex}`);
  };

  const onActionSelection = async (action: string, script: IHostScript) => {
    switch (action) {
      case "showDetails":
        if (!script.last_execution) return;
        onShowDetails(script.last_execution.execution_id);
        break;
      case "run":
        try {
          await scriptsAPI.runScript({
            host_id: host.id,
            script_id: script.script_id,
          });
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

  if (isErrorScriptData) {
    return <DataError card />;
  }
  const scriptColumnConfigs = generateTableColumnConfigs(onActionSelection);
  const data = generateDataSet(
    currentUser,
    host,
    hostScriptResponse?.scripts || []
  );

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
          columns={scriptColumnConfigs}
          data={data}
          isLoading={isLoadingScriptData}
          onQueryChange={onQueryChange}
          disableNextPage={hostScriptResponse?.meta.has_next_results}
          defaultPageIndex={page}
          disableCount
        />
      )}
    </Card>
  );
};

export default Scripts;
