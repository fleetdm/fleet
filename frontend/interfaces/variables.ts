export interface IVariable {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface IVariableFormData {
  name: string;
  value: string;
}
