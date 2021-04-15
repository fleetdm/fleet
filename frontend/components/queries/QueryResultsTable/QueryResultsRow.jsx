import React, { Component } from "react";
import { isEqual, omit } from "lodash";

import queryResultInterface from "interfaces/query_result";

class QueryResultsRow extends Component {
  static propTypes = {
    queryResult: queryResultInterface.isRequired,
  };

  shouldComponentUpdate(nextProps) {
    return !isEqual(this.props.queryResult, nextProps.queryResult);
  }

  render() {
    const { queryResult } = this.props;
    const { host_hostname: hostHostname } = queryResult;
    const queryColumns = omit(queryResult, ["host_hostname"]);

    return (
      <tr>
        <td>{hostHostname}</td>
        {Object.keys(queryColumns).map((col) => {
          return <td key={col}>{queryColumns[col]}</td>;
        })}
      </tr>
    );
  }
}

export default QueryResultsRow;
