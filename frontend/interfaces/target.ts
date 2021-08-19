import PropTypes from "prop-types";
import hostInterface, { IHost } from "interfaces/host";
import labelInterface, { ILabel } from "interfaces/label";
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
