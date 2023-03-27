import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";
import { UserRole } from "./user";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  invited_by: PropTypes.number,
  email: PropTypes.string,
  name: PropTypes.string,
  sso_enabled: PropTypes.bool,
  global_role: PropTypes.any, // eslint-disable-line react/forbid-prop-types
  teams: PropTypes.arrayOf(teamInterface),
});

export interface IInvite {
  created_at: string;
  updated_at: string;
  id: number;
  invited_by: number;
  email: string;
  name: string;
  sso_enabled: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
}

export interface ICreateInviteFormData {
  email: string;
  global_role: UserRole | null;
  invited_by?: number;
  name: string;
  sso_enabled?: boolean;
  teams: ITeam[];
}

export interface IEditInviteFormData {
  currentUserId?: number;
  email?: string;
  global_role: UserRole | null;
  name?: string;
  password: null;
  sso_enabled: boolean;
  teams?: ITeam[];
}
