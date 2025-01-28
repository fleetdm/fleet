import React from "react";

import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";
import Icon from "components/Icon";
import { TableSchemaPlatform } from "interfaces/osquery_table";

interface IPLatformListItemProps {
  platform: TableSchemaPlatform;
}

const baseClassListItem = "platform-list-item";

const PlatformListItem = ({ platform }: IPLatformListItemProps) => {
  return (
    <li key={platform} className={baseClassListItem}>
      <Icon name={platform} />
      <span>{PLATFORM_DISPLAY_NAMES[platform]}</span>
    </li>
  );
};

// TODO: remove when freebsd is removed
type IPlatformsWithFreebsd = TableSchemaPlatform | "freebsd";

interface IQueryTablePlatformsProps {
  platforms: IPlatformsWithFreebsd[];
}

const PLATFORM_ORDER = ["darwin", "windows", "linux", "chrome"];

const baseClass = "query-table-platforms";

const QueryTablePlatforms = ({ platforms }: IQueryTablePlatformsProps) => {
  const platformListItems = platforms
    .filter((platform) => platform !== "freebsd")
    .sort((a, b) => PLATFORM_ORDER.indexOf(a) - PLATFORM_ORDER.indexOf(b))
    .map((platform) => {
      return (
        <PlatformListItem
          key={platform}
          platform={platform as TableSchemaPlatform} // TODO: remove when freebsd is removed
        />
      );
    });

  return (
    <div className={baseClass} data-testid="compatibility">
      <h3>Compatible with</h3>
      <ul className={`${baseClass}__platform-list`}>{platformListItems}</ul>
    </div>
  );
};

export default QueryTablePlatforms;
