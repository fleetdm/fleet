import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { keys, omit } from 'lodash';

import Button from 'components/buttons/Button';
import campaignInterface from 'interfaces/campaign';
import filterArrayByHash from 'utilities/filter_array_by_hash';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import QueryResultsRow from 'components/queries/QueryResultsTable/QueryResultsRow';
import QueryProgressDetails from 'components/queries/QueryProgressDetails';
import Spinner from 'components/loaders/Spinner';

const baseClass = 'query-results-table';

class QueryResultsTable extends Component {
  static propTypes = {
    campaign: campaignInterface.isRequired,
    onExportQueryResults: PropTypes.func,
    onToggleQueryFullScreen: PropTypes.func,
    isQueryFullScreen: PropTypes.bool,
    isQueryShrinking: PropTypes.bool,
    onRunQuery: PropTypes.func.isRequired,
    onStopQuery: PropTypes.func.isRequired,
    queryIsRunning: PropTypes.bool,
    queryTimerMilliseconds: PropTypes.number,
  };

  constructor (props) {
    super(props);

    this.state = {
      resultsFilter: {},
    };
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

  renderTable = () => {
    const {
      renderTableHeaderRow,
      renderTableRows,
    } = this;

    const { queryIsRunning, campaign } = this.props;
    const { query_results: queryResults } = campaign;

    const loading = queryIsRunning && (!queryResults || !queryResults.length);

    if (loading) {
      return <Spinner />;
    }

    return (
      <table className={`${baseClass}__table`}>
        <thead>
          {renderTableHeaderRow()}
        </thead>
        <tbody>
          {renderTableRows()}
        </tbody>
      </table>
    );
  }

  render () {
    const {
      campaign,
      onExportQueryResults,
      isQueryFullScreen,
      isQueryShrinking,
      onToggleQueryFullScreen,
      onRunQuery,
      onStopQuery,
      queryIsRunning,
      queryTimerMilliseconds,
    } = this.props;

    const { renderTable } = this;

    const { hosts_count: hostsCount, query_results: queryResults } = campaign;
    const hasNoResults = !queryIsRunning && (!hostsCount.successful || (!queryResults || !queryResults.length));

    const resultsTableWrapClass = classnames(baseClass, {
      [`${baseClass}--full-screen`]: isQueryFullScreen,
      [`${baseClass}--shrinking`]: isQueryShrinking,
      [`${baseClass}__no-results`]: hasNoResults,
    });

    const toggleFullScreenBtnClass = classnames(`${baseClass}__fullscreen-btn`, {
      [`${baseClass}__fullscreen-btn--active`]: isQueryFullScreen,
    });

    return (
      <div className={resultsTableWrapClass}>
        <header className={`${baseClass}__button-wrap`}>
          {isQueryFullScreen && <QueryProgressDetails
            campaign={campaign}
            onRunQuery={onRunQuery}
            onStopQuery={onStopQuery}
            queryIsRunning={queryIsRunning}
            className={`${baseClass}__full-screen`}
            queryTimerMilliseconds={queryTimerMilliseconds}
          />}

          <Button
            className={toggleFullScreenBtnClass}
            onClick={onToggleQueryFullScreen}
            variant="muted"
          >
            <Icon name={isQueryFullScreen ? 'windowed' : 'fullscreen'} />
          </Button>
          <Button
            className={`${baseClass}__export-btn`}
            onClick={onExportQueryResults}
            variant="link"
          >
            Export
          </Button>
        </header>
        <div className={`${baseClass}__table-wrapper`}>
          {hasNoResults && <em className="no-results-message">No results found</em>}
          {!hasNoResults && renderTable()}
        </div>
      </div>
    );
  }
}

export default QueryResultsTable;
