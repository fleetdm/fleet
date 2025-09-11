import React, { useContext } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { AppContext } from "context/app";

import SideNav from "pages/admin/components/SideNav";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import SETUP_EXPERIENCE_NAV_ITEMS from "./SetupExperienceNavItems";

const baseClass = "setup-experience";

interface ISetupExperienceProps {
  params: Params;
  location: { search: string };
  router: InjectedRouter;
  teamIdForApi: number;
}

const SetupExperience = ({
  params,
  location: { search: queryString },
  router,
  teamIdForApi,
}: ISetupExperienceProps) => {
  const { section } = params;
  const { isPremiumTier } = useContext(AppContext);

  // Not premium shows premium message
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  const DEFAULT_SETTINGS_SECTION = SETUP_EXPERIENCE_NAV_ITEMS[0];

  const currentFormSection =
    SETUP_EXPERIENCE_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p>Customize the end user&apos;s setup experience.</p>
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={SETUP_EXPERIENCE_NAV_ITEMS.map((navItem) => ({
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
    </div>
  );
};

export default SetupExperience;
