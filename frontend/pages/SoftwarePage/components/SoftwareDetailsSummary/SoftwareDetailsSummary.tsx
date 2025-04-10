/**
software/titles/:id > Top section
software/versions/:id > Top section
software/os/:id > Top section
*/

import React from "react";

import { getPathWithQueryParams, QueryParams } from "utilities/url";
import paths from "router/paths";

import DataSet from "components/DataSet";
import LastUpdatedHostCount from "components/LastUpdatedHostCount";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";

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
  const hostCountPath = getPathWithQueryParams(paths.MANAGE_HOSTS, queryParams);

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
                hostCount={
                  <TooltipWrapper tipContent="View all hosts">
                    <CustomLink url={hostCountPath} text={hosts.toString()} />
                  </TooltipWrapper>
                }
                lastUpdatedAt={countsUpdatedAt}
              />
            }
          />
        </dl>
      </dl>
    </div>
  );
};

export default SoftwareDetailsSummary;
