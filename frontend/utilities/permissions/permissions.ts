import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";

export const isSandboxMode = (config: IConfig): boolean => {
  return !!config.sandbox_enabled; // TODO: confirm null/undefined config should treated as false based on final API spec
};

export const isFreeTier = (config: IConfig): boolean => {
  return config.license.tier === "free";
};

export const isPremiumTier = (config: IConfig): boolean => {
  return config.license.tier === "premium";
};

export const isMacMdmEnabledAndConfigured = (config: IConfig): boolean => {
  return Boolean(config.mdm.enabled_and_configured);
};

export const isWindowsMdmEnabledAndConfigured = (config: IConfig): boolean => {
  return Boolean(config.mdm.windows_enabled_and_configured);
};

export const isGlobalAdmin = (user: IUser): boolean => {
  return user.global_role === "admin";
};

export const isGlobalMaintainer = (user: IUser): boolean => {
  return user.global_role === "maintainer";
};

export const isGlobalObserver = (user: IUser): boolean => {
  return (
    user.global_role === "observer" || user.global_role === "observer_plus"
  );
};

export const isOnGlobalTeam = (user: IUser): boolean => {
  return user.global_role !== null;
};

// This checks against a specific team
export const isTeamObserver = (
  user: IUser | null,
  teamId: number | null
): boolean => {
  const userTeamRole = user?.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "observer" || userTeamRole === "observer_plus";
};

export const isTeamMaintainer = (
  user: IUser | null,
  teamId: number | null
): boolean => {
  const userTeamRole = user?.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "maintainer";
};

export const isTeamAdmin = (
  user: IUser | null,
  teamId: number | null
): boolean => {
  const userTeamRole = user?.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "admin";
};

const isTeamMaintainerOrTeamAdmin = (
  user: IUser | null,
  teamId: number | null
): boolean => {
  const userTeamRole = user?.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "admin" || userTeamRole === "maintainer";
};

// This checks against all teams
const isAnyTeamObserverPlus = (user: IUser): boolean => {
  if (!isOnGlobalTeam(user)) {
    return user.teams.some((team) => team?.role === "observer_plus");
  }

  return false;
};

const isAnyTeamMaintainer = (user: IUser): boolean => {
  if (!isOnGlobalTeam(user)) {
    return user.teams.some((team) => team?.role === "maintainer");
  }

  return false;
};

const isAnyTeamAdmin = (user: IUser): boolean => {
  if (!isOnGlobalTeam(user)) {
    return user.teams.some((team) => team?.role === "admin");
  }

  return false;
};

const isAnyTeamMaintainerOrTeamAdmin = (user: IUser): boolean => {
  if (!isOnGlobalTeam(user)) {
    return user.teams.some(
      (team) => team?.role === "maintainer" || team?.role === "admin"
    );
  }

  return false;
};

const isOnlyObserver = (user: IUser): boolean => {
  if (isGlobalObserver(user)) {
    return true;
  }

  // Return false if any role is team maintainer or team admin
  if (!isOnGlobalTeam(user)) {
    return !user.teams.some(
      (team) => team?.role === "maintainer" || team?.role === "admin"
    );
  }

  return false;
};

export const isGlobalObserverPlus = (user: IUser): boolean => {
  return user.global_role === "observer_plus";
};

export const isTeamObserverPlus = (
  user: IUser | null,
  teamId: number | null
): boolean => {
  const userTeamRole = user?.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "observer_plus";
};

export const isObserverPlus = (user: IUser, teamId: number | null): boolean => {
  return isGlobalObserverPlus(user) || isTeamObserverPlus(user, teamId);
};

const isNoAccess = (user: IUser): boolean => {
  return user.global_role === null && user.teams.length === 0;
};

export default {
  isSandboxMode,
  isFreeTier,
  isPremiumTier,
  isMacMdmEnabledAndConfigured,
  isWindowsMdmEnabledAndConfigured,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isOnGlobalTeam,
  isTeamObserver,
  isTeamObserverPlus,
  isTeamMaintainer,
  isTeamMaintainerOrTeamAdmin,
  isAnyTeamObserverPlus,
  isAnyTeamMaintainer,
  isAnyTeamMaintainerOrTeamAdmin,
  isTeamAdmin,
  isAnyTeamAdmin,
  isOnlyObserver,
  isObserverPlus,
  isNoAccess,
};
