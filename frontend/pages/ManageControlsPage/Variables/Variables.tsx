import React, { useContext, useEffect, useMemo } from "react";
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

  const defaultSection = navItems[0];
  const matchedSection = navItems.find((item) => item.urlSection === section);
  const currentSection = matchedSection ?? defaultSection;

  // Redirect the bare route (no section) and unknown sections to the default
  // section, preserving the query string (e.g. the ?add_variable deep-link).
  useEffect(() => {
    if (!matchedSection) {
      router.replace(`${defaultSection.path}${location.search}`);
    }
  }, [matchedSection, defaultSection.path, location.search, router]);

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
          path: `${navItem.path}${location.search}`,
        }))}
        activeItem={currentSection.urlSection}
        CurrentCard={<CurrentCard router={router} location={location} />}
      />
    </div>
  );
};

export default Variables;
