import React, { useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import scriptsAPI, {
  IHostScript,
  IHostScriptsResponse,
} from "services/entities/scripts";
import { IApiError, IError } from "interfaces/errors";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import ScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptDetailsModal";

import { generateDataSet, generateTableHeaders } from "./ScriptsTableConfig";

const baseClass = "host-scripts-section";

interface IScriptsProps {
  hostId?: number;
  isHostOnline: boolean;
}

const Scripts = ({ hostId, isHostOnline }: IScriptsProps) => {
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  // used to track the current script execution id we want to show in the show
  // details modal.
  const scriptExecutionId = useRef<string | null>(null);

  const { renderFlash } = useContext(NotificationContext);

  const { data: scriptsData, isLoading, isError } = useQuery<
    IHostScriptsResponse,
    IError,
    IHostScript[]
  >(["scripts", hostId], () => scriptsAPI.getHostScripts(hostId as number), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: Boolean(hostId),
    select: (res) => res?.scripts,
  });

  if (!hostId) return null;

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
          renderFlash("success", "Script successfully queued!");
        } catch (e) {
          const error = e as AxiosResponse<IApiError>;
          renderFlash("error", error.data.errors[0].reason);
        }
        break;
      default:
    }
  };

  const onCancelScriptDetailsModal = () => {
    setShowScriptDetailsModal(false);
    scriptExecutionId.current = null;
  };

  const scriptHeaders = generateTableHeaders(onActionSelection);
  const data = generateDataSet(scriptsData || [], isHostOnline);

  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      <h2>Scripts</h2>
      <TableContainer
        resultsTitle=""
        emptyComponent={() => (
          <EmptyTable
            header="No Scripts"
            iconName="alert"
            info="There is no scripts to display."
          />
        )}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        columns={scriptHeaders}
        data={data}
        isLoading={isLoading}
        disableCount
      />
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
