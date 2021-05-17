import { IUser } from "interfaces/user";

const isGlobalAdmin = (user: IUser) => {
  return user.global_role === "admin";
};

const isGlobalMaintainer = (user: IUser) => {
  return user.global_role === "maintainer";
};

const isOnGlobalTeam = (user: IUser) => {
  return user.global_role !== null;
};

export default {
  isGlobalAdmin,
  isGlobalMaintainer,
  isOnGlobalTeam,
};
