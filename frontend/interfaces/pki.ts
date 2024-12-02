export interface IPkiCert {
  name: string;
  sha256: string;
  not_valid_after: string;
}

export interface IPkiTemplate {
  profile_id: string;
  name: string;
  common_name: string;
  san: { user_principal_names: string[] };
  seat_id: string;
}

export interface IPkiConfig {
  pki_name: string;
  templates: IPkiTemplate[];
}
