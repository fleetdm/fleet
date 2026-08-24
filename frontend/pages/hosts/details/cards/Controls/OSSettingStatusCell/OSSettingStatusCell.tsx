import React from "react";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";
import { getControlDisplayOption } from "../statusDisplayConfig";

const baseClass = "os-settings-status-cell";

interface IOSSettingStatusCellProps {
  profile: IHostMdmProfileWithAddedStatus;
}

const OSSettingStatusCell = ({ profile }: IOSSettingStatusCellProps) => {
  const displayOption = getControlDisplayOption(profile);

  // graceful error - this state should not be reached based on the API spec
  if (!displayOption) {
    return <TextCell value="Unrecognized" />;
  }

  const { statusText, iconName } = displayOption;

  return (
    <span className={baseClass}>
      <Icon name={iconName} />
      <span className={`${baseClass}__status-text`}>{statusText}</span>
    </span>
  );
};

export default OSSettingStatusCell;
