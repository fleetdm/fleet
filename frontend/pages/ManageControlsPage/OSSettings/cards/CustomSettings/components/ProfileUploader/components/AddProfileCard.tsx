import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import ProfileGraphic from "./AddProfileGraphic";

const AddProfileCard = ({
  baseClass,
  setShowModal,
}: {
  baseClass: string;
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}) => (
  <Card color="gray" className={`${baseClass}__card`}>
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
