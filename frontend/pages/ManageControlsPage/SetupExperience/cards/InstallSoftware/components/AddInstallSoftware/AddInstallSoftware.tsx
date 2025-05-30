import React from "react";

import PATHS from "router/paths";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { ISoftwareTitle } from "interfaces/software";
import LinkWithContext from "components/LinkWithContext";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import {
  getInstallSoftwareDuringSetupCount,
  hasNoSoftwareUploaded,
} from "./helpers";

const baseClass = "add-install-software";

interface IAddInstallSoftwareProps {
  currentTeamId: number;
  hasManualAgentInstall: boolean;
  softwareTitles: ISoftwareTitle[] | null;
  onAddSoftware: () => void;
}

const AddInstallSoftware = ({
  currentTeamId,
  hasManualAgentInstall,
  softwareTitles,
  onAddSoftware,
}: IAddInstallSoftwareProps) => {
  const noSoftwareUploaded = hasNoSoftwareUploaded(softwareTitles);
  const installSoftwareDuringSetupCount = getInstallSoftwareDuringSetupCount(
    softwareTitles
  );

  const getAddedText = () => {
    if (noSoftwareUploaded) {
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

    return installSoftwareDuringSetupCount === 0 ? (
      "No software added."
    ) : (
      <>
        {installSoftwareDuringSetupCount} software will be{" "}
        <TooltipWrapper tipContent="Software order will vary.">
          installed during setup
        </TooltipWrapper>
        .
      </>
    );
  };

  const getButtonText = () => {
    if (noSoftwareUploaded) {
      return "Add software";
    }

    return installSoftwareDuringSetupCount === 0
      ? "Add software"
      : "Show selected software";
  };

  const addedText = getAddedText();
  const buttonText = getButtonText();
  const manuallyInstallTooltipText = (
    <>
      Disabled because you manually install Fleet&apos;s agent (
      <b>Bootstrap package {">"} Advanced options</b>). Use your bootstrap
      package to install software during the setup experience.
    </>
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__description-container`}>
        <p className={`${baseClass}__description`}>
          Install software on hosts that automatically enroll to Fleet.
        </p>
        <CustomLink
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-assistant`}
          text="Learn how"
        />
      </div>
      <span className={`${baseClass}__added-text`}>{addedText}</span>
      <div>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              className={`${baseClass}__manual-install-tooltip`}
              tipContent={manuallyInstallTooltipText}
              disableTooltip={disableChildren || !hasManualAgentInstall}
              position="top"
              showArrow
              underline={false}
            >
              <Button
                className={`${baseClass}__button`}
                onClick={onAddSoftware}
                disabled={
                  disableChildren || hasManualAgentInstall || noSoftwareUploaded
                }
              >
                {buttonText}
              </Button>
            </TooltipWrapper>
          )}
        />
      </div>
    </div>
  );
};

export default AddInstallSoftware;
