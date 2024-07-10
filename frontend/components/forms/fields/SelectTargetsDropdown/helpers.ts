// import { IHost } from "interfaces/host";
// import { ILabel } from "interfaces/label";
import {
  ISelectTargetsEntity,
  ISelectHost,
  ISelectLabel,
  ISelectTeam,
} from "interfaces/target";

// export interface IBaseTarget {
//   id: number;
//   created_at: string;
//   updated_at: string;
//   name: string;
//   description?: string;
//   display_text: string;
// }

// export interface ITargetHost extends IBaseTarget, IHost {
//   target_type: string;
// }

// export interface ITargetLabel extends IBaseTarget, ILabel {
//   target_type: string;
// }

// export interface ITargetTeam extends IBaseTarget {
//   target_type: string;
//   count: number;
// }

// export type ITarget = ITargetLabel | ITargetTeam | ITargetHost;

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
