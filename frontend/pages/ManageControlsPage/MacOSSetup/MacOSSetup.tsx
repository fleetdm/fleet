import React, { useContext } from "react";
import PATHS from "router/paths";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { AppContext } from "context/app";

import SideNav from "pages/admin/components/SideNav";
import Button from "components/buttons/Button/Button";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import EmptyTable from "components/EmptyTable";

import MAC_OS_SETUP_NAV_ITEMS from "./MacOSSetupNavItems";
import TurnOnMdmMessage from "../components/TurnOnMdmMessage";

const baseClass = "macos-setup";

interface ISetupEmptyState {
  router: InjectedRouter;
}

const SetupEmptyState = ({ router }: ISetupEmptyState) => {
  const onClickEmptyConnect = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  return (
    <EmptyTable
      header="Setup experience for macOS hosts"
      info="Connect Fleet to the Apple Business Manager to get started."
      primaryButton={
        <Button variant="brand" onClick={onClickEmptyConnect}>
          Connect
        </Button>
      }
    />
  );
};

interface IMacOSSetupProps {
  params: Params;
  location: { search: string };
  router: any;
  teamIdForApi: number;
}

const MacOSSetup = ({
  params,
  location: { search: queryString },
  router,
  teamIdForApi,
}: IMacOSSetupProps) => {
  const { section } = params;
  const { isPremiumTier, config } = useContext(AppContext);

  // MDM is not on so show messaging for user to enable it.
  if (!config?.mdm.enabled_and_configured) {
    return <TurnOnMdmMessage router={router} />;
  }
  // User has not set up Apple Business Manager.
  if (isPremiumTier && !config?.mdm.apple_bm_enabled_and_configured) {
    return <SetupEmptyState router={router} />;
  }

  const DEFAULT_SETTINGS_SECTION = MAC_OS_SETUP_NAV_ITEMS[0];

  const currentFormSection =
    MAC_OS_SETUP_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p>
        Customize the setup experience for hosts that automatically enroll to
        this team.
      </p>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <SideNav
          className={`${baseClass}__side-nav`}
          navItems={MAC_OS_SETUP_NAV_ITEMS.map((navItem) => ({
            ...navItem,
            path: navItem.path.concat(queryString),
          }))}
          activeItem={currentFormSection.urlSection}
          CurrentCard={
            <CurrentCard
              key={teamIdForApi}
              currentTeamId={teamIdForApi}
              router={router}
            />
          }
        />
      )}
    </div>
  );
};

export default MacOSSetup;
