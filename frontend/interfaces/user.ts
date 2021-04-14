import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

export default PropTypes.shape({
  admin: PropTypes.bool,
  email: PropTypes.string,
  enabled: PropTypes.bool,
  force_password_reset: PropTypes.bool,
  gravatarURL: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  position: PropTypes.string,
  username: PropTypes.string,
  teams: PropTypes.arrayOf(teamInterface),
});

export interface IUser {
  admin: boolean;
  email: string;
  enabled: boolean;
  force_password_reset: boolean;
  gravatarURL: string;
  id: number;
  name: string;
  position: string;
  username: string;
  teams: ITeam[];
  global_role: string | null;
}
