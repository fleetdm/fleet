import PropTypes from "prop-types";

export default PropTypes.shape({
  // TODO
});

export interface IActivity {
  id: number;
  created_at: string;
  actor_full_name: string;
  actor_id: number;
  type: string;
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
