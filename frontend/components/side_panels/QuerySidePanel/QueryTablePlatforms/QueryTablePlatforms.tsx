import React from "react";

import { IOsqueryPlatform } from "interfaces/platform";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";
import Icon from "components/Icon";

interface IPLatformListItemProps {
  platform: IOsqueryPlatform;
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
type IPlatformsWithFreebsd = IOsqueryPlatform | "freebsd";

interface IQueryTablePlatformsProps {
  platforms: IPlatformsWithFreebsd[];
}

const baseClass = "query-table-platforms";

const QueryTablePlatforms = ({ platforms }: IQueryTablePlatformsProps) => {
  const platformListItems = platforms
    .filter((platform) => platform !== "freebsd")
    .map((platform) => {
      return (
        <PlatformListItem
          key={platform}
          platform={platform as IOsqueryPlatform} // TODO: remove when freebsd is removed
        />
      );
    });

  return (
    <div className={baseClass}>
      <h3>Compatible with</h3>
      <ul className={`${baseClass}__platform-list`}>{platformListItems}</ul>
    </div>
  );
};

export default QueryTablePlatforms;
