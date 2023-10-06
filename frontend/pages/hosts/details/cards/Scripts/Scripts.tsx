import React from "react";
import { useQuery } from "react-query";

import scriptsAPI, {
  IHostScript,
  IHostScriptsResponse,
} from "services/entities/scripts";
import { IError } from "interfaces/errors";

import Card from "components/Card";
import TableContainer from "components/TableContainer";
import { generateTableHeaders } from "./ScriptsTableConfig";

const baseClass = "host-scripts-section";

interface IScriptsProps {
  hostId?: number;
}

const Scripts = ({ hostId }: IScriptsProps) => {
  const { data, isLoading, isError } = useQuery<
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

  const scriptHeaders = generateTableHeaders();

  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      <h2>Scripts</h2>
      {data && (
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
      )}
    </Card>
  );
};

export default Scripts;
