export interface ISelfServiceCategory {
  id: number;
  name: string;
  fleet_id: number;
  created_at: string;
  updated_at: string;
}

export interface ICreateSelfServiceCategoryFormData {
  fleet_id: number;
  name: string;
}

export interface IEditSelfServiceCategoryFormData {
  name: string;
}
