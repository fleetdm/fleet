export const LEARN_MORE_CALENDARS =
  "https://www.fleetdm.com/learn-more-about/google-workspace-service-accounts";

export const LEARN_MORE_UPGRADE = "https://www.fleetdm.com/upgrade";

export interface ICalendarsFormProps {
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
