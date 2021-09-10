import React from "react";
import { omit } from "lodash";

import { ICampaignQueryResult } from "interfaces/campaign";

interface IQueryResultsRowProps {
  queryResult: ICampaignQueryResult;
}

const QueryResultsRow = ({ queryResult }: IQueryResultsRowProps) => {
  const { host_hostname: hostHostname } = queryResult;
  const queryColumns: any = omit(queryResult, ["host_hostname"]);

  return (
    <tr>
      <td>{hostHostname}</td>
      {Object.keys(queryColumns).map((col) => {
        return <td key={col}>{queryColumns[col]}</td>;
      })}
    </tr>
  );
};

export default React.memo(QueryResultsRow);
