export interface IEnforcementProfile {
  profile_uuid: string;
  team_id: number | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface IEnforcementProfilesResponse {
  profiles: IEnforcementProfile[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}
