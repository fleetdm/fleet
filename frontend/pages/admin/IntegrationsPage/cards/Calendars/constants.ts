import { IConfig } from "interfaces/config";

export const DEFAULT_TRANSPARENCY_URL = "https://fleetdm.com/transparency";

export interface ICalendarsFormProps {
  // todo
  isPremiumTier?: boolean;
  isUpdatingSettings?: boolean;
  handleSubmit: any;
}

export interface IFormField {
  name: string;
  value: string | boolean | number;
}

export interface ICalendarsFormErrors {
  email?: string | null;
  domain?: string | null;
  privateKey?: string | null;
}
