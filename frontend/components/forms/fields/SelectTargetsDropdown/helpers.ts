import {
  ISelectTargetsEntity,
  ISelectHost,
  ISelectLabel,
  ISelectTeam,
} from "interfaces/target";

export const isTargetHost = (
  target: ISelectTargetsEntity
): target is ISelectHost => {
  return target.target_type === "hosts";
};

export const isTargetLabel = (
  target: ISelectTargetsEntity
): target is ISelectLabel => {
  return target.target_type === "labels";
};

export const isTargetTeam = (
  target: ISelectTargetsEntity
): target is ISelectTeam => {
  return target.target_type === "teams";
};
