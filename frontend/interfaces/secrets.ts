export interface ISecret {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface ISecretPayload {
  name: string;
  value: string;
}
