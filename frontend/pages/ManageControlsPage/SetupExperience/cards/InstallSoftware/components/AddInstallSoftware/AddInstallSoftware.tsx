import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

const baseClass = "add-install-software";

interface IAddInstallSoftwareProps {
  noSoftware: boolean;
  selectedSoftwareIds: number[];
  onAddSoftware: () => void;
}

const AddInstallSoftware = ({
  noSoftware,
  selectedSoftwareIds,
  onAddSoftware,
}: IAddInstallSoftwareProps) => {
  const addedText =
    selectedSoftwareIds.length === 0
      ? "No software added."
      : `${selectedSoftwareIds.length} software will be installed during setup.`;
  const buttonText = "Add software";

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__description-container`}>
        <p className={`${baseClass}__description`}>
          Install software on hosts that automatically enroll to Fleet.
        </p>
        <CustomLink newTab url="" text="Learn how" />
      </div>
      <span className={`${baseClass}__added-text`}>{addedText}</span>
      <div>
        <Button
          className={`${baseClass}__button`}
          variant="brand"
          onClick={onAddSoftware}
          disabled={noSoftware}
        >
          {buttonText}
        </Button>
      </div>
    </div>
  );
};

export default AddInstallSoftware;
