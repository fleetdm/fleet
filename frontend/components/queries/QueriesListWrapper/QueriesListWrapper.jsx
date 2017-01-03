import React, { Component, PropTypes } from 'react';
import { pull } from 'lodash';

import Button from 'components/buttons/Button';
import helpers from 'components/queries/QueriesListWrapper/helpers';
import InputField from 'components/forms/fields/InputField';
import NumberPill from 'components/NumberPill';
import QueriesList from 'components/queries/QueriesList';
import queryInterface from 'interfaces/query';

const baseClass = 'queries-list-wrapper';

class QueriesListWrapper extends Component {
  static propTypes = {
    onRemoveScheduledQueries: PropTypes.func,
    onScheduledQueryFormSubmit: PropTypes.func,
    scheduledQueries: PropTypes.arrayOf(queryInterface),
  };

  constructor (props) {
    super(props);

    this.state = {
      querySearchText: '',
      selectAll: false,
      selectedScheduledQueryIDs: [],
    };
  }

  onRemoveScheduledQueries = (evt) => {
    evt.preventDefault();

    const { onRemoveScheduledQueries: handleRemoveScheduledQueries } = this.props;
    const { selectedScheduledQueryIDs } = this.state;

    this.setState({ selectedScheduledQueryIDs: [] });

    return handleRemoveScheduledQueries(selectedScheduledQueryIDs);
  }

  onSelectAllQueries = (shouldSelectAll) => {
    if (shouldSelectAll) {
      const allScheduledQueries = this.getQueries();
      const selectedScheduledQueryIDs = allScheduledQueries.map(sq => sq.id);

      this.setState({ selectedScheduledQueryIDs });

      return false;
    }

    this.setState({ selectedScheduledQueryIDs: [] });

    return false;
  }

  onSelectQuery = (shouldAddQuery, scheduledQueryID) => {
    const { selectedScheduledQueryIDs } = this.state;
    const newSelectedScheduledQueryIDs = shouldAddQuery ?
      selectedScheduledQueryIDs.concat(scheduledQueryID) :
      pull(selectedScheduledQueryIDs, scheduledQueryID);

    this.setState({ selectedScheduledQueryIDs: newSelectedScheduledQueryIDs });

    return false;
  }

  onUpdateQuerySearchText = (querySearchText) => {
    this.setState({ querySearchText });
  }

  getQueries = () => {
    const { scheduledQueries } = this.props;
    const { querySearchText } = this.state;

    return helpers.filterQueries(scheduledQueries, querySearchText);
  }

  renderButton = () => {
    const { onRemoveScheduledQueries } = this;
    const { selectedScheduledQueryIDs } = this.state;

    const scheduledQueryCount = selectedScheduledQueryIDs.length;

    if (scheduledQueryCount) {
      const queryText = scheduledQueryCount === 1 ? 'Query' : 'Queries';

      return (
        <Button
          className={`${baseClass}__query-btn`}
          onClick={onRemoveScheduledQueries}
          variant="alert"
        >
          Remove {queryText}
        </Button>
      );
    }

    return false;
  }

  renderQueryCount = () => {
    const { scheduledQueries } = this.props;
    const queryCount = scheduledQueries.length;
    const queryText = queryCount === 1 ? 'Query' : 'Queries';

    return <h1 className={`${baseClass}__query-count`}><NumberPill number={queryCount} /> {queryText}</h1>;
  }

  renderQueriesList = () => {
    const { getQueries, onHidePackForm, onSelectAllQueries, onSelectQuery } = this;
    const { onScheduledQueryFormSubmit, scheduledQueries } = this.props;
    const { selectedScheduledQueryIDs } = this.state;

    return (
      <div className={`${baseClass}__queries-list-wrapper`}>
        <QueriesList
          onHidePackForm={onHidePackForm}
          onScheduledQueryFormSubmit={onScheduledQueryFormSubmit}
          onSelectAllQueries={onSelectAllQueries}
          onSelectQuery={onSelectQuery}
          scheduledQueries={getQueries()}
          selectedScheduledQueryIDs={selectedScheduledQueryIDs}
          isScheduledQueriesAvailable={!!scheduledQueries.length}
        />
      </div>
    );
  }

  render () {
    const { onUpdateQuerySearchText, renderButton, renderQueryCount, renderQueriesList } = this;
    const { querySearchText } = this.state;

    return (
      <div className={`${baseClass} body-wrap`}>
        {renderQueryCount()}
        <div className={`${baseClass}__query-list-action`}>
          <InputField
            inputWrapperClass={`${baseClass}__search-queries-input`}
            name="search-queries"
            onChange={onUpdateQuerySearchText}
            placeholder="Search Queries"
            value={querySearchText}
          />
          {renderButton()}
        </div>
        {renderQueriesList()}
      </div>
    );
  }
}

export default QueriesListWrapper;
