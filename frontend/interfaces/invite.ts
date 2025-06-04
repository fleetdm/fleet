import { ITeam } from "./team";
import { UserRole } from "./user";

export interface IInvite {
  created_at: string;
  updated_at: string;
  id: number;
  invited_by: number;
  email: string;
  name: string;
  sso_enabled: boolean;
  mfa_enabled?: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
  api_only?: boolean;
}

export interface ICreateInviteFormData {
  email: string;
  global_role: UserRole | null;
  invited_by?: number;
  name: string;
  sso_enabled?: boolean;
  mfa_enabled?: boolean;
  teams: ITeam[];
}

export interface IEditInviteFormData {
  currentUserId?: number;
  email?: string;
  global_role: UserRole | null;
  name?: string;
  password: null;
  sso_enabled: boolean;
  mfa_enabled?: boolean;
  teams?: ITeam[];
}
