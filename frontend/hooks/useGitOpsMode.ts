import { useContext } from "react";

import { AppContext } from "context/app";
import { IConfig, IGitOpsExceptions } from "interfaces/config";

interface UseGitOpsModeResult {
  gitOpsModeEnabled: boolean;
  repoURL?: string;
}

/**
 * The hook's rule, for callers that need it per entity outside a component
 * (e.g. deriving state for a list of actions) rather than once per render.
 */
export const isGitOpsModeEnabledFor = (
  config: IConfig | null,
  entity?: keyof IGitOpsExceptions
): boolean => {
  const enabled = !!config?.gitops?.gitops_mode_enabled;
  const excepted = entity ? !!config?.gitops?.exceptions?.[entity] : false;
  return enabled && !excepted;
};

/**
 * Returns whether GitOps mode is effectively enabled for a given entity,
 * accounting for per-entity exceptions. When an entity is excepted, GitOps
 * mode is treated as disabled for that entity.
 *
 * Call without an argument for global GitOps mode status (e.g. nav indicator).
 */
const useGitOpsMode = (
  entity?: keyof IGitOpsExceptions
): UseGitOpsModeResult => {
  const { config } = useContext(AppContext);
  return {
    gitOpsModeEnabled: isGitOpsModeEnabledFor(config, entity),
    repoURL: config?.gitops?.repository_url,
  };
};

export default useGitOpsMode;
