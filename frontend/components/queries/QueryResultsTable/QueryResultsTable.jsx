import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { keys, omit } from 'lodash';

import Button from 'components/buttons/Button';
import campaignInterface from 'interfaces/campaign';
import filterArrayByHash from 'utilities/filter_array_by_hash';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import QueryResultsRow from 'components/queries/QueryResultsTable/QueryResultsRow';

const baseClass = 'query-results-table';

class QueryResultsTable extends Component {
  static propTypes = {
    campaign: campaignInterface.isRequired,
    onExportQueryResults: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = { resultsFilter: {} };
  }

  onFilterAttribute = (attribute) => {
    return (value) => {
      const { resultsFilter } = this.state;

      this.setState({
        resultsFilter: {
          ...resultsFilter,
          [attribute]: value,
        },
      });

      return false;
    };
  }

  onSetActiveColumn = (activeColumn) => {
    return () => {
      this.setState({ activeColumn });
    };
  }

  renderTableHeaderRowData = (column, index) => {
    const filterable = column === 'hostname' ? 'host_hostname' : column;
    const { activeColumn, resultsFilter } = this.state;
    const { onFilterAttribute, onSetActiveColumn } = this;
    const filterIconClassName = classnames(`${baseClass}__filter-icon`, {
      [`${baseClass}__filter-icon--is-active`]: activeColumn === column,
    });

    return (
      <th key={`query-results-table-header-${index}`}>
        <span><Icon className={filterIconClassName} name="filter" />{column}</span>
        <InputField
          name={column}
          onChange={onFilterAttribute(filterable)}
          onFocus={onSetActiveColumn(column)}
          value={resultsFilter[filterable]}
        />
      </th>
    );
  }

  renderTableHeaderRow = () => {
    const { campaign } = this.props;
    const { renderTableHeaderRowData } = this;
    const { query_results: queryResults } = campaign;

    const queryAttrs = omit(queryResults[0], ['host_hostname']);
    const queryResultColumns = keys(queryAttrs);

    return (
      <tr>
        {renderTableHeaderRowData('hostname', -1)}
        {queryResultColumns.map((column, i) => {
          return renderTableHeaderRowData(column, i);
        })}
      </tr>
    );
  }

  renderTableRows = () => {
    const { campaign } = this.props;
    const { query_results: queryResults } = campaign;
    const { resultsFilter } = this.state;
    const filteredQueryResults = filterArrayByHash(queryResults, resultsFilter);

    return filteredQueryResults.map((queryResult, index) => {
      return (
        <QueryResultsRow
          index={index}
          key={`qrtr-${index}`}
          queryResult={queryResult}
        />
      );
    });
  }

  render () {
    const { campaign, onExportQueryResults } = this.props;
    const {
      renderTableHeaderRow,
      renderTableRows,
    } = this;
    const { hosts_count: hostsCount } = campaign;

    if (!hostsCount || !hostsCount.total) {
      return false;
    }

    if (!hostsCount.successful) {
      return (
        <div className={`${baseClass} ${baseClass}__no-results`}>
          <em>No results found</em>
        </div>
      );
    }

    return (
      <div className={baseClass}>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={onExportQueryResults}
          variant="link"
        >
          Export
        </Button>
        <div className={`${baseClass}__table-wrapper`}>
          <table className={`${baseClass}__table`}>
            <thead>
              {renderTableHeaderRow()}
            </thead>
            <tbody>
              {renderTableRows()}
            </tbody>
          </table>
        </div>
      </div>
    );
  }
}

export default QueryResultsTable;
