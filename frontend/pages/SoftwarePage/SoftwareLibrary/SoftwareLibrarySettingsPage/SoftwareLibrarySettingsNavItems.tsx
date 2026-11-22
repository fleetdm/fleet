import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import PATHS from "router/paths";

import PatchDeadlineSection from "./sections/PatchDeadlineSection";
import SelfServiceCategoriesSection from "./sections/SelfServiceCategoriesSection";
import { ILibrarySettingsCardProps } from "./types";

const getLibrarySettingsNavItems = (): ISideNavItem<ILibrarySettingsCardProps>[] => [
  {
    title: "Self-service categories",
    urlSection: "categories",
    path: PATHS.SOFTWARE_LIBRARY_SETTINGS_CATEGORIES,
    Card: SelfServiceCategoriesSection,
  },
  {
    title: "Patch deadline",
    urlSection: "patch-deadline",
    path: PATHS.SOFTWARE_LIBRARY_SETTINGS_PATCH_DEADLINE,
    Card: PatchDeadlineSection,
  },
];

export default getLibrarySettingsNavItems;
