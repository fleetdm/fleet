import React from "react";

import Icon from "components/Icon";

const baseClass = "empty-reports";

interface IEmptyReportsProps {
  isSearching: boolean;
}

const EmptyReports = ({ isSearching }: IEmptyReportsProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <Icon name="search" color="ui-fleet-black-25" size="extra-large" />
      <h2 className={`${baseClass}__heading`}>
        {isSearching
          ? "No reports match the current search criteria"
          : "No reports for this host"}
      </h2>
      <p className={`${baseClass}__subheading`}>
        Expecting to see reports? Check back later.
      </p>
    </div>
  );
};

export default EmptyReports;
