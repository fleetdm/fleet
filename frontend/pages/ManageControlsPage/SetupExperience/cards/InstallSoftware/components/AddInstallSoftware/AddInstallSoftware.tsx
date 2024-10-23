import React from "react";

import PATHS from "router/paths";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { ISoftwareTitle } from "interfaces/software";
import LinkWithContext from "components/LinkWithContext";

const baseClass = "add-install-software";

interface IAddInstallSoftwareProps {
  currentTeamId: number;
  softwareTitles: ISoftwareTitle[];
  onAddSoftware: () => void;
}

const AddInstallSoftware = ({
  currentTeamId,
  softwareTitles,
  onAddSoftware,
}: IAddInstallSoftwareProps) => {
  const hasNoSoftware = softwareTitles.length === 0;
  const installDuringSetupCount = softwareTitles.filter(
    (software) => software.software_package?.install_during_setup
  ).length;

  let addedText = <></>;
  let buttonText = "";

  if (hasNoSoftware) {
    addedText = (
      <>
        No software available to add. Please{" "}
        <LinkWithContext
          to={PATHS.SOFTWARE_ADD_FLEET_MAINTAINED}
          currentQueryParams={{ team_id: currentTeamId }}
          withParams={{ type: "query", names: ["team_id"] }}
        >
          upload software
        </LinkWithContext>{" "}
        to be able to add during setup experience.{" "}
      </>
    );
    buttonText = "Add software";
  } else if (installDuringSetupCount === 0) {
    addedText = <>No software added.</>;
    buttonText = "Add software";
  } else {
    addedText = (
      <>{installDuringSetupCount} software will be installed during setup.</>
    );
    buttonText = "Show selected software";
  }

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
