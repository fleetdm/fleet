/**
software/titles/:id > Top section
software/versions/:id > Top section
software/os/:id > Top section
*/

import React from "react";

import { QueryParams } from "utilities/url";

import ViewAllHostsLink from "components/ViewAllHostsLink";
import DataSet from "components/DataSet";
import LastUpdatedHostCount from "components/LastUpdatedHostCount";

import SoftwareIcon from "../icons/SoftwareIcon";

const baseClass = "software-details-summary";

interface ISoftwareDetailsSummaryProps {
  title: string;
  type?: string;
  hosts: number;
  countsUpdatedAt?: string;
  /** The query param that will be added when user clicks on "View all hosts" link */
  queryParams: QueryParams;
  name?: string;
  source?: string;
  versions?: number;
  iconUrl?: string;
}

const SoftwareDetailsSummary = ({
  title,
  type,
  hosts,
  countsUpdatedAt,
  queryParams,
  name,
  source,
  versions,
  iconUrl,
}: ISoftwareDetailsSummaryProps) => {
  return (
    <div className={baseClass}>
      <SoftwareIcon name={name} source={source} url={iconUrl} size="xlarge" />
      <dl className={`${baseClass}__info`}>
        <h1>{title}</h1>
        <dl className={`${baseClass}__description-list`}>
          {!!type && <DataSet title="Type" value={type} />}

          {!!versions && <DataSet title="Versions" value={versions} />}
          <DataSet
            title="Hosts"
            value={
              <LastUpdatedHostCount
                hostCount={hosts === 0 ? "---" : hosts}
                lastUpdatedAt={countsUpdatedAt}
              />
            }
          />
        </dl>
      </dl>
      <div>
        <ViewAllHostsLink
          queryParams={queryParams}
          className={`${baseClass}__hosts-link`}
        />
      </div>
    </div>
  );
};

export default SoftwareDetailsSummary;
