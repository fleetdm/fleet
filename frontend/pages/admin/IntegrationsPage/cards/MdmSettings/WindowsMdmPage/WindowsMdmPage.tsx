import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { SingleValue } from "react-select-5";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackButton from "components/BackButton";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import { notify } from "components/ToastNotification";

import { getErrorMessage } from "./helpers";

const baseClass = "windows-mdm-page";

const UNASSIGNED_FLEET = "";

interface ISetWindowsMdmOptions {
  enableMdm: boolean;
  enableAutoMigration: boolean;
  turnOnProgrammatically: boolean;
  defaultFleet: string;
  router: InjectedRouter;
}

const useSetWindowsMdm = ({
  enableMdm,
  enableAutoMigration,
  turnOnProgrammatically,
  defaultFleet,
  router,
}: ISetWindowsMdmOptions) => {
  const { setConfig, isPremiumTier } = useContext(AppContext);

  const updateWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.updateMDMConfig(
        {
          enable_turn_on_windows_mdm_manually:
            enableMdm && !turnOnProgrammatically,
          windows_enabled_and_configured: enableMdm,
          windows_migration_enabled: enableAutoMigration,
          // The default fleet for user-driven enrollment is Premium only; the
          // backend rejects it otherwise.
          ...(isPremiumTier && {
            windows_enrollment: { default_fleet: defaultFleet },
          }),
        },
        true
      );
      setConfig(updatedConfig);
      notify.success("Windows MDM settings successfully updated.");
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
    }

    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  return updateWindowsMdm;
};

interface IWindowsMdmPageProps {
  router: InjectedRouter;
}

const WindowsMdmPage = ({ router }: IWindowsMdmPageProps) => {
  const { config, isPremiumTier, availableTeams } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const [mdmOn, setMdmOn] = useState(
    config?.mdm?.windows_enabled_and_configured ?? false
  );
  const [autoMigration, setAutoMigration] = useState(
    config?.mdm?.windows_migration_enabled ?? false
  );
  const [turnOnProgrammatically, setTurnOnProgrammatically] = useState(
    !(config?.mdm?.enable_turn_on_windows_mdm_manually ?? false)
  );
  const [defaultFleet, setDefaultFleet] = useState(
    config?.mdm?.windows_enrollment?.default_fleet ?? UNASSIGNED_FLEET
  );

  const isConnectedToEntra = !!config?.mdm?.windows_entra_tenant_ids?.length;

  const updateWindowsMdm = useSetWindowsMdm({
    enableMdm: mdmOn,
    enableAutoMigration: autoMigration,
    turnOnProgrammatically,
    defaultFleet,
    router,
  });

  const onChangeMdmOn = () => {
    setMdmOn(!mdmOn);
    // Turning MDM on defaults to programmatic enrollment; turning it off also
    // turns off auto migration.
    !mdmOn ? setTurnOnProgrammatically(true) : setAutoMigration(false);
  };

  const onChangeTurnOnProgrammatically = () => {
    // Auto migration only applies to programmatic enrollment.
    turnOnProgrammatically && setAutoMigration(false);
    setTurnOnProgrammatically(!turnOnProgrammatically);
  };

  const onChangeAutoMigration = () => {
    setAutoMigration(!autoMigration);
  };

  const onChangeDefaultFleet = (option: SingleValue<CustomOptionType>) => {
    setDefaultFleet(option?.value ?? UNASSIGNED_FLEET);
  };

  const onSaveMdm = () => {
    updateWindowsMdm();
  };

  const fleetOptions: CustomOptionType[] = [
    { label: "Unassigned", value: UNASSIGNED_FLEET },
    // Exclude the synthetic "All teams" (-1) and "No team" (0) context entries;
    // "Unassigned" above is the explicit no-fleet choice.
    ...(availableTeams ?? [])
      .filter((t) => t.id > 0)
      .map((t) => ({ label: t.name, value: t.name })),
  ];

  const defaultFleetDropdown = (
    <DropdownWrapper
      name="default-fleet"
      label="Default fleet"
      options={fleetOptions}
      value={defaultFleet}
      onChange={onChangeDefaultFleet}
      isDisabled={!mdmOn || !isConnectedToEntra || gitOpsModeEnabled}
      helpText={
        <>
          New hosts enrolled into MDM are automatically assigned to this fleet.{" "}
          <CustomLink
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/windows-default-fleet"
            newTab
          />
        </>
      }
    />
  );

  const programmaticToggleLabel = (
    <TooltipWrapper
      tipContent={
        <>
          When enabled, MDM is turned on when Fleet&apos;s agent is installed.
          When disabled, end users turn on MDM manually in{" "}
          <b>Settings &gt; Access work or school</b> (requires Microsoft Entra).
          Only applies to manual enrollment.{" "}
          <CustomLink
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/mdm-enrollment"
            newTab
            variant="tooltip-link"
          />
        </>
      }
    >
      Turn on MDM programmatically
    </TooltipWrapper>
  );

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text="Back to MDM"
            path={PATHS.ADMIN_INTEGRATIONS_MDM}
            className={`${baseClass}__back-to-mdm`}
          />
        </div>
        <h1>Windows MDM</h1>
        <form>
          <Slider
            value={mdmOn}
            activeText="Windows MDM on"
            inactiveText="Windows MDM off"
            onChange={onChangeMdmOn}
            disabled={gitOpsModeEnabled}
          />
          {isPremiumTier && (
            <Slider
              value={turnOnProgrammatically}
              activeText={programmaticToggleLabel}
              inactiveText={programmaticToggleLabel}
              onChange={onChangeTurnOnProgrammatically}
              disabled={!mdmOn || gitOpsModeEnabled}
            />
          )}
          {isPremiumTier && (
            <div className={`${baseClass}__section`}>
              <h2 className={`${baseClass}__section-title`}>
                User driven enrollment
              </h2>
              {isConnectedToEntra ? (
                defaultFleetDropdown
              ) : (
                <TooltipWrapper
                  tipContent={
                    <>
                      Fleet must be connected to Entra to set a default fleet.{" "}
                      <CustomLink
                        text="Learn more"
                        url={
                          PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS
                        }
                        variant="tooltip-link"
                      />
                    </>
                  }
                  showArrow
                  underline={false}
                >
                  {defaultFleetDropdown}
                </TooltipWrapper>
              )}
            </div>
          )}
          {isPremiumTier && turnOnProgrammatically && (
            <div className={`${baseClass}__section`}>
              <h2 className={`${baseClass}__section-title`}>Migration</h2>
              <Checkbox
                disabled={!mdmOn || gitOpsModeEnabled}
                value={autoMigration}
                onChange={onChangeAutoMigration}
              >
                Automatically migrate hosts connected to another MDM solution
              </Checkbox>
            </div>
          )}
          <GitOpsModeTooltipWrapper
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button onClick={onSaveMdm} disabled={disableChildren}>
                Save
              </Button>
            )}
          />
        </form>
      </>
    </MainContent>
  );
};

export default WindowsMdmPage;
