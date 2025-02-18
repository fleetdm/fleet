import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import ProfileGraphic from "../AddProfileGraphic";

const baseClass = "add-profile-card";

interface IAddProfileCardProps {
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddProfileCard = ({ setShowModal }: IAddProfileCardProps) => (
  <Card color="gray" className={baseClass}>
    <div className={`${baseClass}__card--content-wrap`}>
      <ProfileGraphic baseClass={baseClass} showMessage />
      <Button
        className={`${baseClass}__card--add-button`}
        variant="brand"
        type="button"
        onClick={() => setShowModal(true)}
      >
        Add profile
      </Button>
    </div>
  </Card>
);

export default AddProfileCard;
