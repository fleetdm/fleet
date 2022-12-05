import React from "react";

import { buildQueryStringFromParams } from "utilities/url";
import Icon from "components/Icon";

interface IEventedTableTagProps {
  selectedTableName: string;
}

const baseClass = "evented-table-tag";

const EventedTableTag = ({ selectedTableName }: IEventedTableTagProps) => {
  const queryString = buildQueryStringFromParams({
    utm_source: "fleet-ui",
    utm_table: `table-${selectedTableName}`,
  });

  return (
    <a
      href={`https://fleetdm.com/guides/osquery-evented-tables-overview?${queryString}`}
      className={baseClass}
      target="__blank"
    >
      <Icon name="calendar-check" />
      <span>EVENTED TABLE</span>
    </a>
  );
};

export default EventedTableTag;
