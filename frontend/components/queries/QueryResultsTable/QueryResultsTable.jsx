import React, { Component } from 'react';
import classnames from 'classnames';
import { get, keys, omit, values } from 'lodash';

import campaignInterface from 'interfaces/campaign';
import filterArrayByHash from 'utilities/filter_array_by_hash';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import ProgressBar from 'components/loaders/ProgressBar';

const baseClass = 'query-results-table';

class QueryResultsTable extends Component {
  static propTypes = {
    campaign: campaignInterface.isRequired,
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

  renderProgressDetails = () => {
    const { campaign } = this.props;
    const totalHostsCount = get(campaign, 'totals.count', 0);
    const totalHostsReturned = get(campaign, 'hosts.length', 0);
    const totalRowsCount = get(campaign, 'query_results.length', 0);

    return (
      <div className={`${baseClass}__progress-details`}>
        <span>
          <b>{totalHostsReturned}</b>&nbsp;of&nbsp;
          <b>{totalHostsCount} Hosts</b>&nbsp;Returning&nbsp;
          <b>{totalRowsCount} Records</b>
        </span>
        <ProgressBar max={totalHostsCount} value={totalHostsReturned} />
      </div>
    );
  }

  renderTableHeaderRowData = (column, index) => {
    const { onFilterAttribute, onSetActiveColumn } = this;
    const { activeColumn, resultsFilter } = this.state;
    const filterIconClassName = classnames(`${baseClass}__filter-icon`, {
      [`${baseClass}__filter-icon--is-active`]: activeColumn === column,
    });

    return (
      <th key={`query-results-table-header-${index}`}>
        <span><Icon className={filterIconClassName} name="filter" />{column}</span>
        <InputField
          name={column}
          onChange={onFilterAttribute(column)}
          onFocus={onSetActiveColumn(column)}
          value={resultsFilter[column]}
        />
      </th>
    );
  }

  renderTableHeaderRow = () => {
    const { campaign } = this.props;
    const { renderTableHeaderRowData } = this;
    const { query_results: queryResults } = campaign;

    const queryAttrs = omit(queryResults[0], ['hostname']);
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

    return filteredQueryResults.map((row, index) => {
      const queryAttrs = omit(row, ['hostname']);
      const queryResult = values(queryAttrs);

      return (
        <tr key={`query-results-table-row-${index}`}>
          <td>{row.hostname}</td>
          {queryResult.map((attribute, i) => {
            return <td key={`query-results-table-row-data-${i}`}>{attribute}</td>;
          })}
        </tr>
      );
    });
  }

  render () {
    const { campaign } = this.props;
    const {
      renderProgressDetails,
      renderTableHeaderRow,
      renderTableRows,
    } = this;
    const { query_results: queryResults } = campaign;

    if (!queryResults || !queryResults.length) {
      return false;
    }

    return (
      <div className={baseClass}>
        {renderProgressDetails()}
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
