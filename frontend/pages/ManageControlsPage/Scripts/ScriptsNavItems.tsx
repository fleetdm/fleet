import { useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import ScriptBatchProgress, {
  IScriptBatchProgressProps,
} from "./cards/ScriptBatchProgress/ScriptBatchProgress";
import ScriptLibrary, {
  IScriptLibraryProps,
} from "./cards/ScriptLibrary/ScriptLibrary";
import { ScriptsLocation } from "./Scripts";

export interface IScriptsCommonProps {
  router: InjectedRouter;
  location: ScriptsLocation;
  teamId: number;
}
/** no distinct props for each card as of now, keeping this as is for easy future updates to any cards  */
type IScriptsCardProps = IScriptLibraryProps | IScriptBatchProgressProps;

const useScriptNavItems = (
  teamId: number | undefined
): ISideNavItem<IScriptsCardProps>[] => {
  const { isTeamTechnician } = useContext(AppContext);

  return useMemo(() => {
    const items: ISideNavItem<IScriptsCardProps>[] = [
      {
        title: "Library",
        urlSection: "library",
        path: `${PATHS.CONTROLS_SCRIPTS_LIBRARY}?team_id=${teamId || 0}`,
        Card: ScriptLibrary,
      },
    ];

    if (!isTeamTechnician) {
      items.push({
        title: "Batch progress",
        urlSection: "progress",
        path: `${PATHS.CONTROLS_SCRIPTS_BATCH_PROGRESS}?team_id=${teamId || 0}`,
        Card: ScriptBatchProgress,
      });
    }

    return items;
  }, [teamId, isTeamTechnician]);
};

export default useScriptNavItems;
