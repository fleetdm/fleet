import { IHost } from "interfaces/host";
import { ILabel, ILabelSummary } from "interfaces/label";
import { ITeam } from "interfaces/team";

export type ITarget = IHost | ILabel | ITeam;

export interface ISelectHost extends IHost {
  target_type?: string;
}

export interface ISelectLabel extends ILabelSummary {
  target_type?: string;
  display_text?: string;
  query?: string;
  count?: number;
}

export interface ISelectTeam extends ITeam {
  target_type?: string;
  display_text?: string;
}

export type ISelectTargetsEntity = ISelectHost | ISelectLabel | ISelectTeam;

export interface ISelectedTargetsForApi {
  hosts: number[];
  labels: number[];
  fleets: number[];
}

export interface ISelectedTargetsByType {
  hosts: IHost[];
  labels: ILabel[];
  fleets: ITeam[];
}

// TODO: Also use for testing
export const DEFAULT_TARGETS: ITarget[] = [];

export const DEFAULT_TARGETS_BY_TYPE: ISelectedTargetsByType = {
  hosts: [],
  labels: [],
  fleets: [],
};
