import React, { useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";

import SideNav from "pages/admin/components/SideNav";
import PageDescription from "components/PageDescription";

import getVariablesNavItems from "./VariablesNavItems";

const baseClass = "variables";

interface IVariablesProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { add_variable?: string };
  };
}

const Variables = ({ router, params, location }: IVariablesProps) => {
  const { section } = params;

  const { isPremiumTier } = useContext(AppContext);

  const navItems = useMemo(() => getVariablesNavItems(), []);

  const DEFAULT_SECTION = navItems[0];

  const currentSection =
    navItems.find((item) => item.urlSection === section) ?? DEFAULT_SECTION;

  // Redirect the bare route (no section) and unknown sections to the default
  // section, preserving the query string (e.g. the ?add_variable deep-link).
  if (
    !section ||
    (currentSection === DEFAULT_SECTION &&
      section !== DEFAULT_SECTION.urlSection)
  ) {
    router.replace(DEFAULT_SECTION.path.concat(location.search));
    return null;
  }

  const CurrentCard = currentSection.Card;

  return (
    <div className={baseClass}>
      <PageDescription
        variant="tab-panel"
        content={
          isPremiumTier
            ? "Add global variables and custom host vitals to use in scripts and configuration profiles for all fleets."
            : "Add global variables and custom host vitals to use in scripts and configuration profiles."
        }
      />
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={navItems.map((navItem) => ({
          ...navItem,
          path: navItem.path.concat(location.search),
        }))}
        activeItem={currentSection.urlSection}
        CurrentCard={<CurrentCard router={router} location={location} />}
      />
    </div>
  );
};

export default Variables;
