import React, { useRef, useState } from "react";
import { useQuery } from "react-query";

import scriptsAPI, {
  IHostScript,
  IHostScriptsResponse,
} from "services/entities/scripts";
import { IError } from "interfaces/errors";

import Card from "components/Card";
import TableContainer from "components/TableContainer";
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

  if (!hostId || !scriptsData) return null;

  const onActionSelection = (action: string, script: IHostScript): void => {
    switch (action) {
      case "showDetails":
        if (!script.last_execution) return;
        scriptExecutionId.current = script.last_execution.execution_id;
        setShowScriptDetailsModal(true);
        break;
      case "run":
        console.log("running");
        break;
      default:
    }
  };

  const onCancelScriptDetailsModal = () => {
    setShowScriptDetailsModal(false);
    scriptExecutionId.current = null;
  };

  const scriptHeaders = generateTableHeaders(onActionSelection);
  const data = generateDataSet(scriptsData, isHostOnline);

  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      <h2>Scripts</h2>
      <TableContainer
        resultsTitle=""
        emptyComponent={() => <span>No scripts</span>}
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
