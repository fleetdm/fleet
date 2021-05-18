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

export default {
  isCoreTier,
  isBasicTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isOnGlobalTeam,
};
