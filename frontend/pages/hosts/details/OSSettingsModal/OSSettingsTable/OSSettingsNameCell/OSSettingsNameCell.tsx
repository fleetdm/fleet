import React from "react";

import { ProfileScope } from "interfaces/mdm";

import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "os-settings-name-cell";

interface IOSSettingsNameCellProps {
  profileName: string;
  scope: ProfileScope | null;
  managedAccount: string | null;
}

const OSSettingsNameCell = ({
  profileName,
  scope,
  managedAccount,
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
              <b>{managedAccount}</b>
            </>
          }
          position="top"
          underline={false}
          showArrow
        >
          <Icon name="user" />
        </TooltipWrapper>
      )}
    </div>
  );
};

export default OSSettingsNameCell;
