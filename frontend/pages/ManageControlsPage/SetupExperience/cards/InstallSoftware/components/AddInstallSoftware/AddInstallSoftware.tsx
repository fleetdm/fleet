import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { ISoftwareTitle } from "interfaces/software";

const baseClass = "add-install-software";

interface IAddInstallSoftwareProps {
  softwareTitles: ISoftwareTitle[];
  onAddSoftware: () => void;
}

const AddInstallSoftware = ({
  softwareTitles,
  onAddSoftware,
}: IAddInstallSoftwareProps) => {
  const hasNoSoftware = softwareTitles.length === 0;
  const hasSelectedSoftware = softwareTitles.some(
    (software) => software.install_during_setup
  );

  const addedText =
    hasNoSoftware || !hasSelectedSoftware
      ? "No software added."
      : `${
          softwareTitles.filter((software) => software.install_during_setup)
            .length
        } software will be installed during setup.`;
  const buttonText =
    !hasNoSoftware && !hasSelectedSoftware
      ? "Add software"
      : "Show selected software";

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
          disabled={hasNoSoftware}
        >
          {buttonText}
        </Button>
      </div>
    </div>
  );
};

export default AddInstallSoftware;
