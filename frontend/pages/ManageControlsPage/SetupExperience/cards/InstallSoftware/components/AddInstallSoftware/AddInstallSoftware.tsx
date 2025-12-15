import React, { useContext, useState } from "react";
import { capitalize } from "lodash";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { SetupExperiencePlatform } from "interfaces/platform";
import { ISoftwareTitle } from "interfaces/software";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import RevealButton from "components/buttons/RevealButton";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import {
  getInstallSoftwareDuringSetupCount,
  hasNoSoftwareUploaded,
} from "./helpers";

const baseClass = "add-install-software";

const getPlatformLabel = (platform: SetupExperiencePlatform) => {
  switch (platform) {
    case "macos":
      return "macOS";
    case "ios":
      return "iOS";
    case "ipados":
      return "iPadOS";
    default:
      return capitalize(platform);
  }
};

const getAddSoftwareUrl = (
  platform: SetupExperiencePlatform,
  teamId: number
) => {
  let path = "";
  switch (platform) {
    case "ios":
    case "ipados":
    case "android":
      path = PATHS.SOFTWARE_ADD_APP_STORE;
      break;
    case "linux":
      path = PATHS.SOFTWARE_ADD_PACKAGE;
      break;
    default:
      path = PATHS.SOFTWARE_ADD_FLEET_MAINTAINED;
  }

  const params = {
    team_id: teamId,
    // Add android param to preselect Android dropdown on the Add App store page
    ...(platform === "android" && { platform }),
  };

  return `${path}?${buildQueryStringFromParams(params)}`;
};

interface IAddInstallSoftwareProps {
  currentTeamId: number;
  hasManualAgentInstall: boolean;
  softwareTitles: ISoftwareTitle[] | null;
  onAddSoftware: () => void;
  platform: SetupExperiencePlatform;
  savedRequireAllSoftwareMacOS?: boolean | null;
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

  const renderAddedText = () => {
    if (noSoftwareUploaded) {
      return (
        <>
          No {getPlatformLabel(platform)} software available. You can add
          software on the{" "}
          <CustomLink
            url={getAddSoftwareUrl(platform, currentTeamId)}
            text="Software page"
          />
          .
        </>
      );
    }

    const orderTooltip =
      platform === "android"
        ? "Software order will vary."
        : "Installation order will depend on software name, starting with 0-9 then A-Z.";

    return installSoftwareDuringSetupCount === 0 ? (
      "No software selected."
    ) : (
      <>
        {installSoftwareDuringSetupCount} software item
        {installSoftwareDuringSetupCount > 1 && "s"} will be{" "}
        <TooltipWrapper tipContent={orderTooltip}>
          installed during setup
        </TooltipWrapper>
        .
      </>
    );
  };

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
      </div>
      <span className={`${baseClass}__added-text`}>{renderAddedText()}</span>
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
