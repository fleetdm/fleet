import PropTypes from "prop-types";
import hostInterface, { IHost } from "interfaces/host";
import labelInterface, { ILabel, ILabelSummary } from "interfaces/label";
import teamInterface, { ITeam } from "interfaces/team";

export default PropTypes.oneOfType([
  hostInterface,
  labelInterface,
  teamInterface,
]);

export type ITarget = IHost | ILabel | ITeam;
export interface ITargets {
  hosts: IHost[];
  labels: ILabel[];
  teams: ITeam[];
}

export interface ITargetsAPIResponse {
  targets: ITargets;
  targets_count: number;
  targets_missing_in_action: number;
  targets_offline: number;
  targets_online: number;
}

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
  teams: number[];
}

export interface ISelectedTargetsByType {
  hosts: IHost[];
  labels: ILabel[];
  teams: ITeam[];
}

export interface IPackTargets {
  host_ids: (number | string)[];
  label_ids: (number | string)[];
  team_ids: (number | string)[];
}

// TODO: Also use for testing
export const DEFAULT_TARGETS: ITarget[] = [];

export const DEFAULT_TARGETS_BY_TYPE: ISelectedTargetsByType = {
  hosts: [],
  labels: [],
  teams: [],
};
