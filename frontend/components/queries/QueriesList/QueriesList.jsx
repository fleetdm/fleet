import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { includes, size } from 'lodash';

import queryInterface from 'interfaces/query';
import Icon from 'components/icons/Icon';
import QueriesListItem from 'components/queries/QueriesList/QueriesListItem';
import Checkbox from 'components/forms/fields/Checkbox';

const baseClass = 'queries-list';

class QueriesList extends Component {
  static propTypes = {
    isScheduledQueriesAvailable: PropTypes.bool,
    onSelectAllQueries: PropTypes.func.isRequired,
    onSelectQuery: PropTypes.func.isRequired,
    scheduledQueries: PropTypes.arrayOf(queryInterface).isRequired,
    selectedScheduledQueryIDs: PropTypes.arrayOf(PropTypes.number).isRequired,
  };

  constructor (props) {
    super(props);

    this.state = { allQueriesSelected: false };
  }

  isChecked = (scheduledQuery) => {
    const { selectedScheduledQueryIDs } = this.props;
    const { allQueriesSelected } = this.state;

    if (allQueriesSelected) {
      return true;
    }

    return includes(selectedScheduledQueryIDs, scheduledQuery.id);
  }

  handleSelectAllQueries = (shouldSelectAllQueries) => {
    const { onSelectAllQueries } = this.props;
    const { allQueriesSelected } = this.state;

    this.setState({ allQueriesSelected: !allQueriesSelected });

    return onSelectAllQueries(shouldSelectAllQueries);
  }

  renderHelpText = () => {
    const { isScheduledQueriesAvailable, scheduledQueries } = this.props;

    if (scheduledQueries.length) {
      return false;
    }

    if (isScheduledQueriesAvailable) {
      return (
        <tr>
          <td colSpan={6}>
            <p>No queries matched your search criteria.</p>
          </td>
        </tr>
      );
    }

    return (
      <tr>
        <td colSpan={6}>
          <div className={`${baseClass}__first-query`}>
            <h1>First let&apos;s <span>add a query</span>.</h1>
            <h2>Then we&apos;ll set the following:</h2>
            <p><strong>interval:</strong> the amount of time, in seconds, the query waits before running</p>
            <p><strong>platform:</strong> the computer platform where this query will run (other platforms ignored)</p>
            <p><strong>minimum <Icon name="osquery" /> version:</strong> the minimum required <strong>osqueryd</strong> version installed on a host</p>
            <p><strong>logging type:</strong></p>
            <ul>
              <li><strong><Icon name="plus-minus" /> differential:</strong> show only what’s added from last run</li>
              <li><strong><Icon name="bold-plus" /> differential (ignore removals):</strong> show only what’s been added since the last run</li>
              <li><strong><Icon name="camera" /> snapshot:</strong> show everything in its current state</li>
            </ul>
          </div>
        </td>
      </tr>
    );
  }

  render () {
    const { onSelectQuery, scheduledQueries, selectedScheduledQueryIDs } = this.props;
    const { allQueriesSelected } = this.state;
    const { renderHelpText, handleSelectAllQueries } = this;

    const wrapperClassName = classnames(`${baseClass}__table`, {
      [`${baseClass}__table--query-selected`]: size(selectedScheduledQueryIDs),
    });

    return (
      <div className={baseClass}>
        <table className={wrapperClassName}>
          <thead>
            <tr>
              <th><Checkbox
                name="select-all-queries"
                onChange={handleSelectAllQueries}
                value={allQueriesSelected}
              /></th>
              <th>Query Name</th>
              <th>Interval [s]</th>
              <th>Platform</th>
              <th><Icon name="osquery" /> Ver.</th>
              <th>Logging</th>
            </tr>
          </thead>
          <tbody>
            {renderHelpText()}
            {!!scheduledQueries.length && scheduledQueries.map((scheduledQuery) => {
              return (
                <QueriesListItem
                  checked={this.isChecked(scheduledQuery)}
                  key={`scheduled-query-${scheduledQuery.id}`}
                  onSelect={onSelectQuery}
                  scheduledQuery={scheduledQuery}
                />
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default QueriesList;
