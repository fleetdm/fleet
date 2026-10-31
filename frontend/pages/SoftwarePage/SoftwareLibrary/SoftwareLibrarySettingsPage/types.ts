import { InjectedRouter } from "react-router";

export interface ILibrarySettingsLocation {
  pathname: string;
  search: string;
  query: {
    fleet_id?: string;
    team_id?: string;
    add_category?: string;
  };
  hash?: string;
}

// Common prop shape for every card rendered by SoftwareLibrarySettingsPage via
// SideNav. Each section receives everything the Settings shell derived (fleet,
// role, GitOps mode) so it never re-derives shell state. router and location
// are included because SelfServiceCategoriesSection reads add_category=1 from
// the URL and strips it via router.replace.
export interface ILibrarySettingsCardProps {
  router: InjectedRouter;
  location: ILibrarySettingsLocation;
  fleetId: number;
  isRouteOk: boolean;
  canManage: boolean;
  gitOpsModeEnabled: boolean;
}
