import ViewAllHostsLink from "components/ViewAllHostsLink";
import React from "react";

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
  softwareId: number;
  title: string;
  type: string;
  hosts: number;
  version?: string;
}

const SoftwareDetailsSummary = ({
  softwareId,
  title,
  type,
  hosts,
  version,
}: ISoftwareDetailsSummaryProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__icon`}>icon</div>
      <dl className={`${baseClass}__info`}>
        <h1>{title}</h1>
        <dl className={`${baseClass}__description-list`}>
          <DataSet
            title="Type"
            // value={formatSoftwareType(software.source)} TODO: format value
            value={type}
          />
          {version && <DataSet title="Version" value={version} />}
          <DataSet title="Hosts" value={hosts === 0 ? "---" : hosts} />
        </dl>
      </dl>
      <div>
        <ViewAllHostsLink
          queryParams={{ software_id: softwareId }}
          className={`${baseClass}__hosts-link`}
        />
      </div>
    </div>
  );
};

export default SoftwareDetailsSummary;
