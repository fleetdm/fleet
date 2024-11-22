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
  two_factor_authentication_enabled: PropTypes.bool,
  global_role: PropTypes.string,
  api_only: PropTypes.bool,
  teams: PropTypes.arrayOf(teamInterface),
});

export const USERS_ROLES = [
  "admin",
  "maintainer",
  "observer",
  "observer_plus",
] as const;
export type IUserRole = typeof USERS_ROLES[number];
export type UserRole =
  | "admin"
  | "maintainer"
  | "observer"
  | "observer_plus"
  | "gitops"
  | "Admin"
  | "Maintainer"
  | "Observer"
  | "Observer+"
  | "GitOps"
  | "Unassigned"
  | ""
  | "Various";

export interface IUser {
  created_at?: string;
  updated_at?: string;
  id: number;
  name: string;
  email: string;
  role: UserRole;
  force_password_reset: boolean;
  gravatar_url?: string;
  gravatar_url_dark?: string;
  sso_enabled: boolean;
  two_factor_authentication_enabled?: boolean;
  global_role: UserRole | null;
  api_only: boolean;
  teams: ITeam[];
}

/**
 * The shape of the request body when updating a user.
 */
export interface IUserUpdateBody {
  global_role?: UserRole | null;
  teams?: ITeam[];
  name: string;
  email?: string;
  sso_enabled?: boolean;
  two_factor_authentication_enabled?: boolean;
  role?: UserRole;
  id: number;
}

export interface IUserFormErrors {
  email?: string | null;
  name?: string | null;
  password?: string | null;
  sso_enabled?: boolean | null;
}
export interface IResetPasswordFormErrors {
  new_password?: string | null;
  new_password_confirmation?: string | null;
}

export interface IResetPasswordForm {
  new_password: string;
  new_password_confirmation: string;
}

export interface ILoginUserData {
  email: string;
  password: string;
}

export interface ICreateUserFormData {
  email: string;
  global_role: UserRole | null;
  name: string;
  password?: string | null;
  sso_enabled?: boolean;
  two_factor_authentication_enabled?: boolean;
  teams: ITeam[];
}

export interface IUpdateUserFormData {
  currentUserId?: number;
  email?: string;
  global_role?: UserRole | null;
  name?: string;
  password?: string | null;
  sso_enabled?: boolean;
  two_factor_authentication_enabled?: boolean;
  teams?: ITeam[];
}

export interface ICreateUserWithInvitationFormData {
  email: string;
  invite_token: string;
  name: string;
  password: string;
  password_confirmation: string;
}
