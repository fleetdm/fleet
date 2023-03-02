import React from "react";
import { Params } from "react-router/lib/Router";

import SideNav from "pages/admin/components/SideNav";

import MAC_OS_SETTINGS_NAV_ITEMS from "./MacOSSettingsNavItems";
import AggregateMacSettingsIndicators from "./AggregateMacSettingsIndicators";

const baseClass = "mac-os-settings";

interface IMacOSSettingsProps {
  params: Params;
  // location field looks like this to integrate with the react router Route component, which
  // renders this one
  location: {
    query: { team_id?: string };
  };
}

const MacOSSettings = ({ params, location }: IMacOSSettingsProps) => {
  const { section } = params;
  const { team_id } = location.query;
  // Avoids possible case where Number(undefined) returns NaN
  const teamId = team_id === undefined ? team_id : Number(team_id);

  const DEFAULT_SETTINGS_SECTION = MAC_OS_SETTINGS_NAV_ITEMS[0];

  const currentFormSection =
    MAC_OS_SETTINGS_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely enforce settings on macOS hosts assigned to this team.
      </p>
      <AggregateMacSettingsIndicators teamId={teamId} />
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={MAC_OS_SETTINGS_NAV_ITEMS}
        activeItem={currentFormSection.urlSection}
        CurrentCard={<CurrentCard />}
      />
    </div>
  );
};

export default MacOSSettings;
