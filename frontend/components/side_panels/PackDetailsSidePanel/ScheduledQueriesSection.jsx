import React, { Component } from "react";
import PropTypes from "prop-types";
import { Link } from "react-router";

import scheduledQueryInterface from "interfaces/scheduled_query";

const baseClass = "pack-details-side-panel";

class ScheduledQueriesSection extends Component {
  static propTypes = {
    scheduledQueries: PropTypes.arrayOf(scheduledQueryInterface),
  };

  render() {
    const { scheduledQueries } = this.props;

    return (
      <div className={`${baseClass}__scheduled-queries`}>
        <p className={`${baseClass}__section-label`}>Queries</p>

        <ul className={`${baseClass}__queries-list`}>
          {scheduledQueries.map((scheduledQuery) => {
            return (
              <li key={`scheduled-query-${scheduledQuery.id}`}>
                <Link
                  to={`/queries/${scheduledQuery.query_id}`}
                  className={`${baseClass}__query-name`}
                >
                  {scheduledQuery.name}
                </Link>
              </li>
            );
          })}
        </ul>
      </div>
    );
  }
}

export default ScheduledQueriesSection;
