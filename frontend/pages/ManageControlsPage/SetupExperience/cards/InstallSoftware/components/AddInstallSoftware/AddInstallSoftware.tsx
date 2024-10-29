import React from "react";

import PATHS from "router/paths";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { ISoftwareTitle } from "interfaces/software";
import LinkWithContext from "components/LinkWithContext";

const baseClass = "add-install-software";

interface IAddInstallSoftwareProps {
  currentTeamId: number;
  softwareTitles: ISoftwareTitle[] | null;
  onAddSoftware: () => void;
}

const AddInstallSoftware = ({
  currentTeamId,
  softwareTitles,
  onAddSoftware,
}: IAddInstallSoftwareProps) => {
  const hasNoSoftware = !softwareTitles || softwareTitles.length === 0;

  const getAddedText = () => {
    if (hasNoSoftware) {
      return (
        <>
          No software available to add. Please{" "}
          <LinkWithContext
            to={PATHS.SOFTWARE_ADD_FLEET_MAINTAINED}
            currentQueryParams={{ team_id: currentTeamId }}
            withParams={{ type: "query", names: ["team_id"] }}
          >
            upload software
          </LinkWithContext>{" "}
          to be able to add during setup experience.
        </>
      );
    }

    const installDuringSetupCount = softwareTitles.filter(
      (software) =>
        software.software_package?.install_during_setup ||
        software.app_store_app?.install_during_setup
    ).length;

    return installDuringSetupCount === 0
      ? "No software added."
      : `${installDuringSetupCount} software will be installed during setup.`;
  };

  const getButtonText = () => {
    if (hasNoSoftware) {
      return "Add software";
    }

    const installDuringSetupCount = softwareTitles.filter(
      (software) =>
        software.software_package?.install_during_setup ||
        software.app_store_app?.install_during_setup
    ).length;

    return installDuringSetupCount === 0
      ? "Add software"
      : "Show selected software";
  };

  const addedText = getAddedText();
  const buttonText = getButtonText();

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
