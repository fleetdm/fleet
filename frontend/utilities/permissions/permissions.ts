import { IUser } from "interfaces/user";
import { IConfig } from "../../interfaces/config";

const isCoreTier = (config: IConfig): boolean => {
  return config.tier === "core";
};

const isBasicTier = (config: IConfig): boolean => {
  return config.tier === "basic";
};

const isGlobalAdmin = (user: IUser): boolean => {
  return user.global_role === "admin";
};

const isGlobalMaintainer = (user: IUser): boolean => {
  return user.global_role === "maintainer";
};

const isGlobalObserver = (user: IUser): boolean => {
  return user.global_role === "observer";
};

const isOnGlobalTeam = (user: IUser): boolean => {
  return user.global_role !== null;
};

const isTeamObserver = (user: IUser, teamId: number): boolean => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "observer";
};

const isTeamMaintainer = (user: IUser, teamId: number): boolean => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "maintainer";
};

const isOnlyObserver = (user: IUser): boolean => {
  if (isGlobalObserver(user)) {
    return true;
  }

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
  isOnlyObserver,
};
