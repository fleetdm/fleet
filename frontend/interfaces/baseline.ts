export interface IBaselineCategory {
  name: string;
  profiles: string[];
  policies: string[];
  scripts: string[];
}

export interface IBaselineManifest {
  id: string;
  name: string;
  version: string;
  platform: string;
  description: string;
  categories: IBaselineCategory[];
}

export interface IBaselinesResponse {
  baselines: IBaselineManifest[];
}

export interface IApplyBaselineRequest {
  baseline_id: string;
  team_id: number;
}

export interface IApplyBaselineResponse {
  baseline_id: string;
  team_id: number;
  profiles_created: string[];
  policies_created: string[];
  scripts_created: string[];
}
