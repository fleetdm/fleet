import PropTypes from "prop-types";
import teamInterface, { ITeam } from "./team";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  email: PropTypes.string,
  role: PropTypes.string,
  force_password_reset: PropTypes.bool,
  gravatar_url: PropTypes.string,
  sso_enabled: PropTypes.bool,
  global_role: PropTypes.string,
  api_only: PropTypes.bool,
  teams: PropTypes.arrayOf(teamInterface),
});

export interface IUser {
  created_at?: string;
  updated_at?: string;
  id: number;
  name: string;
  email: string;
  role: string;
  force_password_reset: boolean;
  gravatar_url: string;
  sso_enabled: boolean;
  global_role: string | null;
  api_only: boolean;
  teams: ITeam[];
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

export interface IUserFormErrors {
  email: string | null;
  name: string | null;
  password: string | null;
  sso_enabled: boolean | null;
}

export interface ICreateUserFormDataNoInvite {
  email: string;
  global_role: string | null;
  name: string;
  password?: string | null;
  sso_enabled?: boolean | undefined;
  teams: ITeam[];
}

export interface IDeleteSessionsUser {
  actions: { label: string; disabled: boolean; value: string }[];
  email: string;
  id: number;
  name: string;
  roles?: string;
  status: string;
  teams?: string;
  type: string;
}

export interface IDestroyUser {
  actions: { label: string; disabled: boolean; value: string }[];
  email: string;
  id: number;
  name: string;
  roles?: string;
  status: string;
  teams?: string | null;
  type: string;
}

export interface IUpdateUser {
  api_only: boolean;
  created_at: string;
  updated_at: string;
  email?: string;
  force_password_reset: boolean;
  global_role?: string | null;
  gravatarURL?: string;
  gravatar_url: string;
  id: number;
  name: string;
  sso_enabled?: boolean;
  teams: ITeam[];
}

export interface IUpdateUserFormData {
  currentUserId?: number;
  email?: string;
  global_role?: string | null;
  name?: string;
  password?: string | null;
  sso_enabled?: boolean;
  teams?: ITeam[];
}
