import ViewAllHostsLink from "components/ViewAllHostsLink";
import React from "react";
import SoftwareIcon from "../icons/SoftwareIcon";

const baseClass = "software-details-summary";

interface IDescriptionSetProps {
  title: string;
  value: React.ReactNode;
}

// TODO: move to frontend/components
const DataSet = ({ title, value }: IDescriptionSetProps) => {
  return (
    <div className={`${baseClass}__data-set`}>
      <dt>{title}</dt>
      <dd>{value}</dd>
    </div>
  );
};

interface ISoftwareDetailsSummaryProps {
  id: number;
  title: string;
  type?: string;
  hosts: number;
  /** The query param name that will be added when user clicks on "View all hosts" link */
  queryParam: string;
  name?: string;
  source?: string;
  versions?: number;
}

const SoftwareDetailsSummary = ({
  id,
  title,
  type,
  hosts,
  queryParam,
  name,
  source,
  versions,
}: ISoftwareDetailsSummaryProps) => {
  return (
    <div className={baseClass}>
      <SoftwareIcon name={name} source={source} size="large" />
      <dl className={`${baseClass}__info`}>
        <h1>{title}</h1>
        <dl className={`${baseClass}__description-list`}>
          {type && <DataSet title="Type" value={type} />}

          {versions && <DataSet title="Versions" value={versions} />}
          <DataSet title="Hosts" value={hosts === 0 ? "---" : hosts} />
        </dl>
      </dl>
      <div>
        <ViewAllHostsLink
          queryParams={{ [queryParam]: id }}
          className={`${baseClass}__hosts-link`}
        />
      </div>
    </div>
  );
};

export default SoftwareDetailsSummary;
