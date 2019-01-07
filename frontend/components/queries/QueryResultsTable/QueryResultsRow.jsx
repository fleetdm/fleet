import React, { Component } from 'react';
import PropTypes from 'prop-types';
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
    const { host_hostname: hostHostname } = queryResult;
    const queryAttrs = omit(queryResult, ['host_hostname']);
    const queryAttrValues = values(queryAttrs);

    return (
      <tr>
        <td>{hostHostname}</td>
        {queryAttrValues.map((attribute, i) => {
          return <td key={`query-results-table-row-${index}-${i}`}>{attribute}</td>;
        })}
      </tr>
    );
  }
}

export default QueryResultsRow;
