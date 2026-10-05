import React from "react";

import Icon from "components/Icon";
import { buildQueryStringFromParams } from "utilities/url";

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
