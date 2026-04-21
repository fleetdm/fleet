import { useContext } from "react";
import { AppContext } from "context/app";
import { IGitOpsExceptions } from "interfaces/config";

interface UseGitOpsModeResult {
  gitOpsModeEnabled: boolean;
  repoURL?: string;
}

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
  const enabled = !!config?.gitops?.gitops_mode_enabled;
  const excepted = entity ? !!config?.gitops?.exceptions?.[entity] : false;
  return {
    gitOpsModeEnabled: enabled && !excepted,
    repoURL: config?.gitops?.repository_url,
  };
};

export default useGitOpsMode;
