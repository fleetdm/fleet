import { IPolicy } from "./policy";
import { IQuery } from "./query";

export enum ActivityType {
  CreatedPack = "created_pack",
  DeletedPack = "deleted_pack",
  EditedPack = "edited_pack",
  CreatedPolicy = "created_policy",
  DeletedPolicy = "deleted_policy",
  EditedPolicy = "edited_policy",
  CreatedSavedQuery = "created_saved_query",
  DeletedSavedQuery = "deleted_saved_query",
  EditedSavedQuery = "edited_saved_query",
  CreatedTeam = "created_team",
  DeletedTeam = "deleted_team",
  LiveQuery = "live_query",
  AppliedSpecPack = "applied_spec_pack",
  AppliedSpecPolicy = "applied_spec_policy",
  AppliedSpecSavedQuery = "applied_spec_saved_query",
  UserAddedBySSO = "user_added_by_sso",
}
export interface IActivity {
  created_at: string;
  id: number;
  actor_full_name: string;
  actor_id: number;
  actor_gravatar: string;
  actor_email?: string;
  type: ActivityType;
  details?: IActivityDetails;
}
export interface IActivityDetails {
  pack_id?: number;
  pack_name?: string;
  policy_id?: number;
  policy_name?: string;
  query_id?: number;
  query_name?: string;
  team_id?: number;
  team_name?: string;
  targets_count?: number;
  specs?: IQuery[] | IPolicy[];
}
