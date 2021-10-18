import PropTypes from "prop-types";
import { IQuery } from "./query";

export enum ActivityType {
  CreatedPack = "created_pack",
  DeletedPack = "deleted_pack",
  EditedPack = "edited_pack",
  CreatedSavedQuery = "created_saved_query",
  DeletedSavedQuery = "deleted_saved_query",
  EditedSavedQuery = "edited_saved_query",
  CreatedTeam = "created_team",
  DeletedTeam = "deleted_team",
  LiveQuery = "live_query",
  AppliedSpecPack = "applied_spec_pack",
  AppliedSpecSavedQuery = "applied_spec_saved_query",
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
  query_id?: number;
  query_name?: string;
  team_id?: number;
  team_name?: string;
  targets_count?: number;
  specs?: IQuery[];
}

export default PropTypes.shape({
  created_at: PropTypes.string,
  id: PropTypes.number,
  actor_full_name: PropTypes.string,
  actor_id: PropTypes.number,
  actor_gravatar: PropTypes.string,
  actor_email: PropTypes.string,
  type: PropTypes.string,
  details: PropTypes.shape({
    pack_id: PropTypes.number,
    pack_name: PropTypes.string,
    query_id: PropTypes.number,
    query_name: PropTypes.string,
    team_id: PropTypes.number,
    team_name: PropTypes.string,
    targets_count: PropTypes.number,
  }),
});
