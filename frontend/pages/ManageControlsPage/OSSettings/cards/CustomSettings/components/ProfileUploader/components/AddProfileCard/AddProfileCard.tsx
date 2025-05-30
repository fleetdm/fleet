import React from "react";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Card from "components/Card";
import Button from "components/buttons/Button";
import ProfileGraphic from "../AddProfileGraphic";

const baseClass = "add-profile-card";

interface IAddProfileCardProps {
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddProfileCard = ({ setShowModal }: IAddProfileCardProps) => (
  <Card color="grey" className={baseClass}>
    <div className={`${baseClass}__card--content-wrap`}>
      <ProfileGraphic baseClass={baseClass} showMessage />
      <GitOpsModeTooltipWrapper
        tipOffset={8}
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            className={`${baseClass}__card--add-button`}
            type="button"
            onClick={() => setShowModal(true)}
          >
            Add profile
          </Button>
        )}
      />
    </div>
  </Card>
);

export default AddProfileCard;
