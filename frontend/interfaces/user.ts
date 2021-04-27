import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

export default PropTypes.shape({
  admin: PropTypes.bool,
  email: PropTypes.string,
  enabled: PropTypes.bool,
  force_password_reset: PropTypes.bool,
  global_role: PropTypes.string,
  gravatarURL: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  sso_enabled: PropTypes.bool,
  teams: PropTypes.arrayOf(teamInterface),
  username: PropTypes.string,
});

export interface IUser {
  admin: boolean;
  email: string;
  enabled: boolean;
  force_password_reset: boolean;
  global_role: string | null;
  gravatarURL: string;
  id: number;
  name: string;
  sso_enabled: boolean;
  teams: ITeam[];
  username: string;
}

export interface IUserUpdateBody {
  global_role?: string | null;
  teams?: ITeam[];
  name?: string;
  email?: string;
  sso_enabled?: boolean;
}
