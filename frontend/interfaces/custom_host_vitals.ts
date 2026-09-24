export interface ICustomHostVital {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface ICustomHostVitalFormData {
  name: string;
}

// The per-host projection of a custom host vital
export interface IHostCustomVital {
  custom_host_vital_id: number;
  name: string;
  value: string;
}
