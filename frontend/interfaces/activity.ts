import PropTypes from "prop-types";

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
  AppliedPackSpec = "applied_pack_spec",
  AppliedQuerySpec = "applied_query_spec",
}
export interface IActivity {
  id: number;
  created_at: string;
  actor_email?: string;
  actor_full_name: string;
  actor_id: number;
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
}

export default PropTypes.shape({
  id: PropTypes.number,
  created_at: PropTypes.string,
  actor_email: PropTypes.string,
  actor_full_name: PropTypes.string,
  actor_id: PropTypes.number,
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
