import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import React from "react";

const baseClass = "profile-list-heading";

interface IProfileListHeadingProps {
  entityName: string;
  createEntityText: string;
  onClickAdd?: () => void;
}

const ProfileListHeading = ({
  entityName,
  createEntityText,
  onClickAdd,
}: IProfileListHeadingProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__profile-name-heading`}>{entityName}</span>
      <span className={`${baseClass}__actions-heading`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="brand-inverse-icon"
              className={`${baseClass}__add-button`}
              onClick={onClickAdd}
              iconStroke
            >
              <>
                <Icon name="plus" color="core-fleet-green" />
                {createEntityText}
              </>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default ProfileListHeading;
