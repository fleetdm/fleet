import React, { Component } from "react";
import { humanQueryLastRun } from "kolide/helpers";

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

    const secondsToHms = (d) => {
      const h = Math.floor(d / 3600);
      const m = Math.floor((d % 3600) / 60);
      const s = Math.floor((d % 3600) % 60);

      const hDisplay = h > 0 ? h + (h === 1 ? " hr " : " hrs ") : "";
      const mDisplay = m > 0 ? m + (m === 1 ? " min " : " mins ") : "";
      const sDisplay = s > 0 ? s + (s === 1 ? " sec " : " secs ") : "";
      return hDisplay + mDisplay + sDisplay;
    };

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
