import { IListSort } from "./list_options";

export interface IHostCertificate {
  id: number;
  not_valid_after: string;
  not_valid_before: string;
  certificate_authority: boolean;
  common_name: string;
  key_algorithm: string;
  key_strength: number;
  key_usage: string;
  serial: string;
  signing_algorithm: string;
  subject: {
    country: string;
    organization: string;
    organizational_unit: string;
    common_name: string;
  };
  issuer: {
    country: string;
    organization: string;
    organizational_unit: string;
    common_name: string;
  };
  source: string;
  username: string;
}

export const CERTIFICATES_DEFAULT_SORT: IListSort = {
  order_key: "common_name",
  order_direction: "asc",
} as const;
