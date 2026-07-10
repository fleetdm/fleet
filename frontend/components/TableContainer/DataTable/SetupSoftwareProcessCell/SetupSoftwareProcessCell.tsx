import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import React from "react";

const baseClass = "setup-software-process-cell";

interface ISetupSoftwareProcessCell {
  name: string;
  /** Raw software name used for SoftwareIcon fallback matching when url is null.
   * Display-name overrides won't match the known-icon lookup (e.g. FMAs without
   * a custom icon_url), so pass the raw title name here. Defaults to `name`. */
  iconName?: string;
  url?: string | null;
}

const SetupSoftwareProcessCell = ({
  name,
  iconName,
  url,
}: ISetupSoftwareProcessCell) => {
  return (
    <span className={baseClass}>
      <SoftwareIcon name={iconName ?? name ?? ""} size="small" url={url} />
      <div>
        Install <b>{name || "Unknown software"}</b>
      </div>
    </span>
  );
};

export default SetupSoftwareProcessCell;
