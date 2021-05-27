import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";

export const isCoreTier = (config: IConfig): boolean => {
  return config.tier === "core";
};

export const isBasicTier = (config: IConfig): boolean => {
  return config.tier === "basic";
};

export const isGlobalAdmin = (user: IUser): boolean => {
  return user.global_role === "admin";
};

export const isGlobalMaintainer = (user: IUser): boolean => {
  return user.global_role === "maintainer";
};

export const isGlobalObserver = (user: IUser): boolean => {
  return user.global_role === "observer";
};

export const isOnGlobalTeam = (user: IUser): boolean => {
  return user.global_role !== null;
};

const isTeamObserver = (user: IUser, teamId: number): boolean => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "observer";
};

// This checks against a specific team
const isTeamMaintainer = (user: IUser, teamId: number): boolean => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "maintainer";
};

// This checks against all teams
const isAnyTeamMaintainer = (user: IUser): boolean => {
  if (!isOnGlobalTeam(user)) {
    return user.teams.some((team) => team?.role === "maintainer");
  }

  return false;
};

const isOnlyObserver = (user: IUser): boolean => {
  if (isGlobalObserver(user)) {
    return true;
  }

  // Return false if any role is team maintainer
  if (!isOnGlobalTeam(user)) {
    return !user.teams.some((team) => team?.role === "maintainer");
  }

  return false;
};

export default {
  isCoreTier,
  isBasicTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isOnGlobalTeam,
  isTeamObserver,
  isTeamMaintainer,
  isAnyTeamMaintainer,
  isOnlyObserver,
};
