export interface IEnrollSecret {
  secret: string;
  created_at?: string;
  team_id?: number;
}

export interface IEnrollSecretsResponse {
  secrets: IEnrollSecret[];
}
