import React, { Component, PropTypes } from 'react';
import { isEqual, omit, values } from 'lodash';

import queryResultInterface from 'interfaces/query_result';

class QueryResultsRow extends Component {
  static propTypes = {
    index: PropTypes.number.isRequired,
    queryResult: queryResultInterface.isRequired,
  };

  shouldComponentUpdate (nextProps) {
    return !isEqual(this.props.queryResult, nextProps.queryResult);
  }

  render () {
    const { index, queryResult } = this.props;
    const { hostname } = queryResult;
    const queryAttrs = omit(queryResult, ['hostname']);
    const queryAttrValues = values(queryAttrs);

    return (
      <tr>
        <td>{hostname}</td>
        {queryAttrValues.map((attribute, i) => {
          return <td key={`query-results-table-row-${index}-${i}`}>{attribute}</td>;
        })}
      </tr>
    );
  }
}

export default QueryResultsRow;
