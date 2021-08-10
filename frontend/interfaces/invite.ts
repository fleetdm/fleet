import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

// update this entire thing

export default PropTypes.shape({
  admin: PropTypes.bool,
  email: PropTypes.string,
  gravatarURL: PropTypes.string,
  id: PropTypes.number,
  invited_by: PropTypes.number,
  name: PropTypes.string,
  teams: PropTypes.arrayOf(teamInterface),
});

export interface IInvite {
  admin: boolean;
  email: string;
  gravatarURL: string;
  id: number;
  invited_by: number;
  name: string;
  teams: ITeam[];
  sso_enabled: boolean;
  global_role: string | null;
}
