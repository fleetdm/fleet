import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import React from "react";

const baseClass = "profile-list-heading";

interface IProfileListHeadingProps {
  onClickAddProfile?: () => void;
}

const ProfileListHeading = ({
  onClickAddProfile,
}: IProfileListHeadingProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__profile-name-heading`}>
        Configuration profile
      </span>
      <span className={`${baseClass}__actions-heading`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="text-icon"
              className={`${baseClass}__add-button`}
              onClick={onClickAddProfile}
            >
              <span className={`${baseClass}__icon-wrap`}>
                <Icon name="plus" />
                Add profile
              </span>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default ProfileListHeading;
