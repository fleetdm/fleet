import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import ScriptBatchProgress, {
  IScriptBatchProgressProps,
} from "./cards/ScriptBatchProgress/ScriptBatchProgress";
import ScriptLibrary, {
  IScriptLibraryProps,
} from "./cards/ScriptLibrary/ScriptLibrary";

export interface IScriptsCommonProps {
  router: InjectedRouter;
  teamId: number;
}
type IScriptsCardProps = IScriptLibraryProps | IScriptBatchProgressProps;

const SCRIPTS_NAV_ITEMS: ISideNavItem<IScriptsCardProps>[] = [
  {
    title: "Library",
    urlSection: "library",
    path: `${PATHS.CONTROLS_SCRIPTS_LIBRARY}`,
    Card: ScriptLibrary,
  },
  {
    title: "Batch progress",
    urlSection: "progress",
    path: `${PATHS.CONTROLS_SCRIPTS_BATCH_PROGRESS}`,
    Card: ScriptBatchProgress,
  },
];

export default SCRIPTS_NAV_ITEMS;
