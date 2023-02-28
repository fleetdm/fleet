import React from "react";
import { Params } from "react-router/lib/Router";

import SideNav from "pages/admin/components/SideNav";

import MAC_OS_SETTINGS_NAV_ITEMS from "./MacOSSettingsNavItems";

const baseClass = "mac-os-settings";

interface IMacOSSettingsProps {
  params: Params;
}

const MacOSSettings = ({ params }: IMacOSSettingsProps) => {
  const { section } = params;
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
