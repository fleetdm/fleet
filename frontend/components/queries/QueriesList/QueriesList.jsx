import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { includes, size } from 'lodash';

import queryInterface from 'interfaces/query';
import Checkbox from 'components/forms/fields/Checkbox';
import QueriesListRow from 'components/queries/QueriesList/QueriesListRow';

const baseClass = 'queries-list';

class QueriesList extends Component {
  static propTypes = {
    checkedQueryIDs: PropTypes.arrayOf(PropTypes.number).isRequired,
    isQueriesAvailable: PropTypes.bool,
    onCheckAll: PropTypes.func.isRequired,
    onCheckQuery: PropTypes.func.isRequired,
    onSelectQuery: PropTypes.func.isRequired,
    queries: PropTypes.arrayOf(queryInterface).isRequired,
    selectedQuery: queryInterface,
  };

  static defaultProps = {
    selectedQuery: {},
  };

  constructor (props) {
    super(props);

    this.state = { allQueriesChecked: false };
  }

  isChecked = (query) => {
    const { checkedQueryIDs } = this.props;
    const { allQueriesChecked } = this.state;

    if (allQueriesChecked) {
      return true;
    }

    return includes(checkedQueryIDs, query.id);
  }

  handleCheckAll = (shouldCheckAllQueries) => {
    const { onCheckAll } = this.props;
    const { allQueriesChecked } = this.state;

    this.setState({ allQueriesChecked: !allQueriesChecked });

    return onCheckAll(shouldCheckAllQueries);
  }

  renderHelpText = () => {
    const { isQueriesAvailable, queries } = this.props;

    if (queries.length) {
      return false;
    }

    if (isQueriesAvailable) {
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
          <p>No queries available. Try creating one.</p>
        </td>
      </tr>
    );
  }

  render () {
    const { checkedQueryIDs, onCheckQuery, onSelectQuery, queries, selectedQuery } = this.props;
    const { allQueriesChecked } = this.state;
    const { renderHelpText, handleCheckAll } = this;

    const wrapperClassName = classnames(`${baseClass}__table`, {
      [`${baseClass}__table--query-selected`]: size(checkedQueryIDs),
    });

    return (
      <div className={baseClass}>
        <table className={wrapperClassName}>
          <thead>
            <tr>
              <th><Checkbox
                name="check-all-queries"
                onChange={handleCheckAll}
                value={allQueriesChecked}
              /></th>
              <th>Query Name</th>
              <th className={`${baseClass}__author-name`}>Author</th>
              <th>Last Modified</th>
            </tr>
          </thead>
          <tbody>
            {renderHelpText()}
            {!!queries.length && queries.map((query) => {
              return (
                <QueriesListRow
                  checked={this.isChecked(query)}
                  key={`query-row-${query.id}`}
                  onCheck={onCheckQuery}
                  onSelect={onSelectQuery}
                  query={query}
                  selected={selectedQuery.id === query.id}
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

