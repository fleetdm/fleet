import React, { Component } from "react";
import { humanQueryLastRun, secondsToHms } from "kolide/helpers";

import queryInterface from "interfaces/query";

const baseClass = "pack-queries-list-row";

class PackQueriesListRow extends Component {
  static propTypes = {
    query: queryInterface.isRequired,
  };

  render() {
    const { query } = this.props;
    const {
      scheduled_query_name,
      description,
      interval,
      last_executed,
    } = query;

    const frequency = secondsToHms(interval);

    return (
      <tr>
        <td className={`${baseClass}__name`}>{scheduled_query_name}</td>
        <td className={`${baseClass}__description`}>{description}</td>
        <td className={`${baseClass}__frequency`}>{frequency}</td>
        <td className={`${baseClass}__last-run`}>
          {humanQueryLastRun(last_executed)}
        </td>
      </tr>
    );
  }
}

export default PackQueriesListRow;
