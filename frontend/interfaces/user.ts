import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

export default PropTypes.shape({
  email: PropTypes.string,
  force_password_reset: PropTypes.bool,
  api_only: PropTypes.bool,
  global_role: PropTypes.string,
  gravatar_url: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  sso_enabled: PropTypes.bool,
  teams: PropTypes.arrayOf(teamInterface),
  username: PropTypes.string,
});

export interface IUser {
  email: string;
  force_password_reset: boolean;
  api_only: boolean;
  global_role: string | null;
  gravatar_url: string;
  id: number;
  name: string;
  sso_enabled: boolean;
  teams: ITeam[];
  username: string;
}

/**
 * The shape of the request body when updating a user.
 */
export interface IUserUpdateBody {
  global_role?: string | null;
  teams?: ITeam[];
  name?: string;
  email?: string;
  sso_enabled?: boolean;
}
