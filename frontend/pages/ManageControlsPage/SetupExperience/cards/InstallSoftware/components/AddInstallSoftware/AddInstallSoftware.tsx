import React, { useContext, useState } from "react";
import { capitalize } from "lodash";

import PATHS from "router/paths";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { SetupExperiencePlatform } from "interfaces/platform";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { ISoftwareTitle } from "interfaces/software";
import Checkbox from "components/forms/fields/Checkbox";
import LinkWithContext from "components/LinkWithContext";
import RevealButton from "components/buttons/RevealButton";
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
  platform: SetupExperiencePlatform;
  savedRequireAllSoftwareMacOS: boolean | null | undefined;
}

const AddInstallSoftware = ({
  currentTeamId,
  hasManualAgentInstall,
  softwareTitles,
  onAddSoftware,
  platform,
  savedRequireAllSoftwareMacOS,
}: IAddInstallSoftwareProps) => {
  const noSoftwareUploaded = hasNoSoftwareUploaded(softwareTitles);
  const installSoftwareDuringSetupCount = getInstallSoftwareDuringSetupCount(
    softwareTitles
  );
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);
  const [showMacOSOptions, setShowMacOSOptions] = useState(false);
  const [requireAllSoftwareMacOS, setRequireAllSoftwareMacOS] = useState(
    savedRequireAllSoftwareMacOS || false
  );
  const [isUpdating, setIsUpdating] = useState(false);

  // Handle clicking Save button for "Cancel setup if software install fails" option.
  const onClickSave = async () => {
    setIsUpdating(true);
    try {
      await mdmAPI.updateRequireAllSoftwareMacOS(
        currentTeamId,
        requireAllSoftwareMacOS
      );
      renderFlash("success", "Successfully updated!");
    } catch {
      renderFlash("error", "Couldn't update. Please try again.");
    } finally {
      setIsUpdating(false);
    }
  };

  const getAddedText = () => {
    let platformText = "";

    switch (platform) {
      case "macos":
        platformText = "macOS";
        break;
      case "ios":
        platformText = "iOS";
        break;
      case "ipados":
        platformText = "iPadOS";
        break;
      default:
        platformText = capitalize(platform);
    }

    if (noSoftwareUploaded) {
      return (
        <>
          No {platformText} software available. You can add software on the{" "}
          <LinkWithContext
            to={PATHS.SOFTWARE_ADD_FLEET_MAINTAINED}
            currentQueryParams={{ team_id: currentTeamId }}
            withParams={{ type: "query", names: ["team_id"] }}
          >
            Software page
          </LinkWithContext>
          .
        </>
      );
    }

    return installSoftwareDuringSetupCount === 0 ? (
      "No software selected."
    ) : (
      <>
        {installSoftwareDuringSetupCount} software item
        {installSoftwareDuringSetupCount > 1 && "s"} will be{" "}
        <TooltipWrapper tipContent="Software order will vary.">
          installed during setup.
        </TooltipWrapper>
      </>
    );
  };

  const addedText = getAddedText();
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
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-experience/install-software`}
          text="Learn how"
        />
      </div>
      <span className={`${baseClass}__added-text`}>{addedText}</span>
      {!noSoftwareUploaded && (
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
                    disableChildren ||
                    hasManualAgentInstall ||
                    noSoftwareUploaded
                  }
                >
                  Select software
                </Button>
              </TooltipWrapper>
            )}
          />
        </div>
      )}
      {platform === "macos" && (
        <div className={`${baseClass}__macos_options_form`}>
          <RevealButton
            isShowing={showMacOSOptions}
            showText="Show advanced options"
            hideText="Hide advanced options"
            caretPosition="after"
            onClick={() => setShowMacOSOptions(!showMacOSOptions)}
          />
          {showMacOSOptions && (
            <form>
              <Checkbox
                disabled={config?.gitops.gitops_mode_enabled}
                value={requireAllSoftwareMacOS}
                onChange={setRequireAllSoftwareMacOS}
              >
                <TooltipWrapper tipContent="If any software fails, the end user won't be let through, and will see a prompt to contact their IT admin. Remaining software installs will be canceled.">
                  Cancel setup if software install fails
                </TooltipWrapper>
              </Checkbox>
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button
                    disabled={disableChildren}
                    isLoading={isUpdating}
                    onClick={onClickSave}
                  >
                    Save
                  </Button>
                )}
              />
            </form>
          )}
        </div>
      )}
    </div>
  );
};

export default AddInstallSoftware;
