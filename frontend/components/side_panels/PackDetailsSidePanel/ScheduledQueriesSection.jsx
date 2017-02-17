import React, { Component, PropTypes } from 'react';
import { take } from 'lodash';
import { Link } from 'react-router';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import scheduledQueryInterface from 'interfaces/scheduled_query';

const baseClass = 'pack-details-side-panel';
const DEFAULT_NUM_QUERIES = 6;

class ScheduledQueriesSection extends Component {
  static propTypes = {
    scheduledQueries: PropTypes.arrayOf(scheduledQueryInterface),
  };

  constructor (props) {
    super(props);

    this.state = { showAllQueries: false };
  }

  onShowAll = () => {
    this.setState({ showAllQueries: true });

    return false;
  }

  renderShowMoreQueries = () => {
    const { showAllQueries } = this.state;
    const scheduledQueryCount = this.props.scheduledQueries.length;
    const shouldRenderShowMore = !showAllQueries && scheduledQueryCount > DEFAULT_NUM_QUERIES;

    if (shouldRenderShowMore) {
      const { onShowAll } = this;
      const numMoreQueries = scheduledQueryCount - DEFAULT_NUM_QUERIES;
      const queryText = numMoreQueries === 1 ? 'Query' : 'Queries';

      return (
        <div className={`${baseClass}__more-queries-section`}>
          <span>{numMoreQueries} More {queryText}</span>
          <Button onClick={onShowAll} variant="unstyled">SHOW</Button>
        </div>
      );
    }

    return false;
  }

  render () {
    const { renderShowMoreQueries } = this;
    const { scheduledQueries } = this.props;
    const { showAllQueries } = this.state;
    const queriesToRender = showAllQueries ? scheduledQueries : take(scheduledQueries, DEFAULT_NUM_QUERIES);

    return (
      <div className={`${baseClass}__scheduled-queries`}>
        <p className={`${baseClass}__section-label`}>Queries</p>

        <ul className={`${baseClass}__queries-list`}>
          {queriesToRender.map((scheduledQuery) => {
            return (
              <li key={`scheduled-query-${scheduledQuery.id}`}>
                <Icon className={`${baseClass}__query-icon`} name="query" />
                <Link to={`/queries/${scheduledQuery.query_id}`} className={`${baseClass}__query-name`}>{scheduledQuery.name}</Link>
              </li>
            );
          })}
        </ul>
        {renderShowMoreQueries()}
      </div>
    );
  }
}

export default ScheduledQueriesSection;
