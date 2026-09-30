import React from "react";

import Icon from "components/Icon";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { ProfileScope } from "interfaces/mdm";

const baseClass = "os-settings-name-cell";

interface IOSSettingsNameCellProps {
  profileName: string;
  scope: ProfileScope | null;
  managedAccount: string | null;
  hidden: boolean;
  /** My device doesn't flag hidden profiles; the end user opted to see them. */
  isDeviceUser?: boolean;
}

const OSSettingsNameCell = ({
  profileName,
  scope,
  managedAccount,
  hidden,
  isDeviceUser = false,
}: IOSSettingsNameCellProps) => {
  return (
    <div className={baseClass}>
      <TooltipTruncatedTextCell
        value={profileName}
        className={`${baseClass}__name-tooltip`}
      />
      {scope === "user" && (
        <TooltipWrapper
          className={`${baseClass}__scope-tooltip`}
          tipContent={
            <>
              Scoped to local user account:
              <br />
              <strong>{managedAccount}</strong>
            </>
          }
          position="top"
          underline={false}
          showArrow
        >
          <Icon name="user" color="ui-fleet-black-33" />
        </TooltipWrapper>
      )}
      {hidden && !isDeviceUser && (
        <TooltipWrapper
          tipContent="Hidden from end user"
          position="top"
          underline={false}
          showArrow
        >
          <Icon name="eye-slash" color="ui-fleet-black-33" />
        </TooltipWrapper>
      )}
    </div>
  );
};

export default OSSettingsNameCell;
