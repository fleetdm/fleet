import { IConfig } from "interfaces/config";
import { InjectedRouter } from "react-router";

export const DEFAULT_TRANSPARENCY_URL = "https://fleetdm.com/transparency";

export type DeepPartial<T> = T extends object
  ? {
      [P in keyof T]?: DeepPartial<T[P]>;
    }
  : T;

export interface IAppConfigFormProps {
  appConfig: IConfig;
  isPremiumTier?: boolean;
  isUpdatingSettings?: boolean;
  handleSubmit: (formUpdates: DeepPartial<IConfig>) => Promise<boolean>;
  router: InjectedRouter;
  subsection?: string;
}

export const authMethodOptions = [
  { label: "Plain", value: "authmethod_plain" },
  { label: "Cram MD5", value: "authmethod_cram_md5" },
  { label: "Login", value: "authmethod_login" },
];

export const authTypeOptions = [
  { label: "Username and password", value: "authtype_username_password" },
  { label: "None", value: "authtype_none" },
];

export default {
  authMethodOptions,
  authTypeOptions,
};
