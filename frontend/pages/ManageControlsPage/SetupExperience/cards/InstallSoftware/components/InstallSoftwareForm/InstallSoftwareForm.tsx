import React, { useCallback, useContext, useState, useMemo } from "react";
import { isEqual } from "lodash";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { isMacOS, SetupExperiencePlatform } from "interfaces/platform";
import { ISoftwareTitle } from "interfaces/software";
import { INotification } from "interfaces/notification";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import EmptyTable from "components/EmptyTable";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import InstallSoftwareTable from "../InstallSoftwareTable";
import { hasNoSoftwareUploaded } from "./helpers";

const baseClass = "install-software-form";

const manuallyInstallTooltipText = (
  <>
    Disabled because you manually install Fleet&apos;s agent (
    <b>Bootstrap package {">"} Advanced options</b>). Use your bootstrap package
    to install software during the setup experience.
  </>
);

const initializeSelectedSoftwareIds = (softwareTitles: ISoftwareTitle[]) => {
  return softwareTitles.reduce<number[]>((acc, software) => {
    if (
      software.software_package?.install_during_setup ||
      software.app_store_app?.install_during_setup
    ) {
      acc.push(software.id);
    }
    return acc;
  }, []);
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
    fleet_id: teamId,
    // Add android param to preselect Android dropdown on the Add App store page
    ...(platform === "android" && { platform }),
  };

  return `${path}?${buildQueryStringFromParams(params)}`;
};

interface IInstallSoftwareFormProps {
  currentTeamId: number;
  hasManualAgentInstall: boolean;
  softwareTitles: ISoftwareTitle[] | null;
  platform: SetupExperiencePlatform;
  savedRequireAllSoftwareMacOS?: boolean | null;
  router: InjectedRouter;
  refetchSoftwareTitles: () => void;
}

const InstallSoftwareForm = ({
  currentTeamId,
  hasManualAgentInstall,
  softwareTitles,
  platform,
  savedRequireAllSoftwareMacOS,
  router,
  refetchSoftwareTitles,
}: IInstallSoftwareFormProps) => {
  const noSoftwareUploaded = hasNoSoftwareUploaded(softwareTitles);
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);
  const [requireAllSoftwareMacOS, setRequireAllSoftwareMacOS] = useState(
    savedRequireAllSoftwareMacOS ?? false
  );
  const [isSaving, setIsSaving] = useState(false);

  const initialSelectedSoftware = useMemo(
    () => (softwareTitles ? initializeSelectedSoftwareIds(softwareTitles) : []),
    [softwareTitles]
  );

  // Track if the user changed the macOS checkbox since the last save.
  // We don't compare against props here to avoid races with parent refetch timing.
  const [touchedRequireAll, setTouchedRequireAll] = useState(false);

  const handleChangeRequireAll = (value: boolean) => {
    setRequireAllSoftwareMacOS(value);
    setTouchedRequireAll(true);
  };

  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState<number[]>(
    initialSelectedSoftware
  );

  const installSoftwareDuringSetupCount = selectedSoftwareIds.length;

  const onChangeSoftwareSelect = useCallback((select: boolean, id: number) => {
    setSelectedSoftwareIds((prev) => {
      if (select) {
        if (prev.includes(id)) return prev;
        return [...prev, id];
      }
      return prev.filter((selectedId) => selectedId !== id);
    });
  }, []);

  const isSoftwareSelectionDirty = useMemo(
    () =>
      !isEqual(
        selectedSoftwareIds.slice().sort(),
        initialSelectedSoftware.slice().sort()
      ),
    [selectedSoftwareIds, initialSelectedSoftware]
  );

  const shouldUpdateSoftware = isSoftwareSelectionDirty;
  const shouldUpdateRequireAll = platform === "macos" && touchedRequireAll;

  const onClickSave = async (evt: React.FormEvent) => {
    evt.preventDefault();

    if (!softwareTitles) return;

    setIsSaving(true);

    const errorNotifications: INotification[] = [];
    let hadSuccess = false;

    // 1. Software selection update
    if (shouldUpdateSoftware) {
      try {
        await mdmAPI.updateSetupExperienceSoftware(
          platform,
          currentTeamId,
          selectedSoftwareIds
        );
        hadSuccess = true;
        // Still let parent refetch even if the macOS call later fails
      } catch (e) {
        errorNotifications.push({
          id: "update-software",
          alertType: "error",
          isVisible: true,
          // You can make this more specific if you want to inspect `e`
          message: "Couldn't save software. Please try again.",
          persistOnPageChange: false,
        });
      }
    }

    // 2. macOS “require all software” update
    if (shouldUpdateRequireAll) {
      try {
        await mdmAPI.updateRequireAllSoftwareMacOS(
          currentTeamId,
          requireAllSoftwareMacOS
        );
        hadSuccess = true;
        setTouchedRequireAll(false);
      } catch (e) {
        errorNotifications.push({
          id: "update-require-all",
          alertType: "error",
          isVisible: true,
          message:
            "Couldn't update 'Cancel setup if software install fails'. Please try again.",
          persistOnPageChange: false,
        });
      }
    }

    // 3. Render flashes
    if (errorNotifications.length > 0) {
      renderMultiFlash({ notifications: errorNotifications });
    } else if (hadSuccess) {
      renderFlash("success", "Successfully updated.");
    }

    refetchSoftwareTitles();
    setIsSaving(false);
  };

  const renderCustomCount = () => {
    const orderTooltip =
      platform === "android"
        ? "Software order will vary."
        : "Installation order will depend on software name, starting with 0-9 then A-Z.";

    return (
      <div>
        <strong>
          {installSoftwareDuringSetupCount} software item
          {installSoftwareDuringSetupCount !== 1 && "s"}
        </strong>{" "}
        will be{" "}
        <TooltipWrapper tipContent={orderTooltip}>
          installed during setup
        </TooltipWrapper>
        .
      </div>
    );
  };

  const manualAgentInstallBlockingSoftware =
    hasManualAgentInstall && isMacOS(platform);

  const onClickAddSoftware = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    router.push(getAddSoftwareUrl(platform, currentTeamId));
  };

  const renderEmptyState = () => {
    return (
      <EmptyTable
        className={`${baseClass}__empty-table`}
        header="No software available to install"
        primaryButton={
          <Button
            className={`${baseClass}__button`}
            onClick={onClickAddSoftware}
          >
            Add software
          </Button>
        }
      />
    );
  };

  if (noSoftwareUploaded || !softwareTitles) {
    return <div className={baseClass}>{renderEmptyState()}</div>;
  }

  return (
    <div className={baseClass}>
      <form onSubmit={onClickSave}>
        <InstallSoftwareTable
          softwareTitles={softwareTitles}
          onChangeSoftwareSelect={onChangeSoftwareSelect}
          platform={platform}
          renderCustomCount={renderCustomCount}
          manualAgentInstallBlockingSoftware={
            manualAgentInstallBlockingSoftware
          }
        />
        {platform === "macos" && (
          <div className={`${baseClass}__macos_options`}>
            <Checkbox
              disabled={
                config?.gitops.gitops_mode_enabled ||
                manualAgentInstallBlockingSoftware
              }
              value={requireAllSoftwareMacOS}
              onChange={handleChangeRequireAll}
            >
              <TooltipWrapper tipContent="If any software fails, the end user won't be let through, and will see a prompt to contact their IT admin. Remaining software installs will be canceled.">
                Cancel setup if software install fails
              </TooltipWrapper>
            </Checkbox>
          </div>
        )}
        <GitOpsModeTooltipWrapper
          tipOffset={6}
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              className={"select-software-table__manual-install-tooltip"}
              tipContent={manuallyInstallTooltipText}
              disableTooltip={
                disableChildren || !manualAgentInstallBlockingSoftware
              }
              position="top"
              showArrow
              underline={false}
              tipOffset={12}
            >
              <Button
                disabled={
                  disableChildren ||
                  isSaving ||
                  manualAgentInstallBlockingSoftware
                }
                isLoading={isSaving}
                type="submit"
              >
                Save
              </Button>
            </TooltipWrapper>
          )}
        />
      </form>
    </div>
  );
};

export default InstallSoftwareForm;
