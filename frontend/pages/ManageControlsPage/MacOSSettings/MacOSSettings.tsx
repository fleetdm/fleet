import React, { useContext } from "react";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import SideNav from "pages/admin/components/SideNav";
import { useQuery } from "react-query";
import { IMdmProfile, IMdmProfilesResponse } from "interfaces/mdm";
import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import mdmAPI from "services/entities/mdm";

import MAC_OS_SETTINGS_NAV_ITEMS from "./MacOSSettingsNavItems";

const baseClass = "mac-os-settings";

interface IMacOSSettingsProps {
  params: Params;
  location: {
    search: string;
  };
}

const MacOSSettings = ({
  location: { search: queryString },
  params,
}: IMacOSSettingsProps) => {
  const { section } = params;
  const { currentTeam } = useContext(AppContext);

  const teamId =
    currentTeam?.id === undefined || currentTeam.id < APP_CONTEXT_NO_TEAM_ID
      ? API_NO_TEAM_ID // coerce undefined and -1 to 0 for 'No team'
      : currentTeam.id;

  const { data: profiles, refetch: refectchProfiles } = useQuery<
    IMdmProfilesResponse,
    unknown,
    IMdmProfile[] | null
  >(["profiles", teamId], () => mdmAPI.getProfiles(teamId), {
    select: (data) => data.profiles,
    refetchOnWindowFocus: false,
  });

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
      {profiles && <AggregateMacSettingsIndicators teamId={teamId} />}
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={MAC_OS_SETTINGS_NAV_ITEMS.map((navItem) => ({
          ...navItem,
          path: navItem.path.concat(queryString),
        }))}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          <CurrentCard
            key={teamId}
            currentTeamId={teamId}
            profiles={profiles}
            onProfileUpload={refectchProfiles}
            onProfileDelete={refectchProfiles}
          />
        }
      />
    </div>
  );
};

export default MacOSSettings;
