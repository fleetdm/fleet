import React from "react";

import { IOsqueryPlatform } from "interfaces/platform";
import { PLATFORM_DISPLAY_NAMES, PLATFORM_ICONS } from "utilities/constants";

interface IPlatformIconProps {
  platform: IOsqueryPlatform;
}

const PlatformIcon = ({ platform }: IPlatformIconProps) => {
  const iconSrc = PLATFORM_ICONS[platform];
  return <img src={iconSrc} alt={`${platform} icon`} className={"icon"} />;
};

interface IPLatformListItemProps {
  platform: IOsqueryPlatform;
}

const baseClassListItem = "platform-list-item";

const PlatformListItem = ({ platform }: IPLatformListItemProps) => {
  return (
    <li key={platform} className={baseClassListItem}>
      <PlatformIcon platform={platform} />
      <span>{PLATFORM_DISPLAY_NAMES[platform]}</span>
    </li>
  );
};

interface IQueryTablePlatformsProps {
  platforms: IOsqueryPlatform[];
}

const baseClass = "query-table-platforms";

const QueryTablePlatforms = ({ platforms }: IQueryTablePlatformsProps) => {
  const platformListItems = platforms.map((platform) => {
    return <PlatformListItem platform={platform} />;
  });

  return (
    <div className={baseClass}>
      <h3>Compatible with</h3>
      <ul className={`${baseClass}__platform-list`}>{platformListItems}</ul>
    </div>
  );
};

export default QueryTablePlatforms;
