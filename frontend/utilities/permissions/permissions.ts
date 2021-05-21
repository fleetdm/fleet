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

const isOnGlobalTeam = (user: IUser): boolean => {
  return user.global_role !== null;
};

const isTeamObserver = (user: IUser, teamId: number) => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "observer";
};

const isTeamMaintainer = (user: IUser, teamId: number) => {
  const userTeamRole = user.teams.find((team) => team.id === teamId)?.role;
  return userTeamRole === "maintainer";
};

export default {
  isCoreTier,
  isBasicTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isOnGlobalTeam,
  isTeamObserver,
  isTeamMaintainer,
};
