import React, { createContext, useReducer, useMemo, ReactNode } from "react";

import { IConfig, IUserUISettings } from "interfaces/config";
import { IEnrollSecret } from "interfaces/enroll_secret";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  ITeamSummary,
  APP_CONTEX_NO_TEAM_SUMMARY,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import { IUser } from "interfaces/user";
import permissions from "utilities/permissions";
import sort from "utilities/sort";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";

enum ACTIONS {
  SET_AVAILABLE_TEAMS = "SET_AVAILABLE_TEAMS",
  SET_UI_SETTINGS = "SET_UI_SETTINGS",
  SET_CURRENT_USER = "SET_CURRENT_USER",
  SET_CURRENT_TEAM = "SET_CURRENT_TEAM",
  SET_CONFIG = "SET_CONFIG",
  SET_ENROLL_SECRET = "SET_ENROLL_SECRET",
  SET_ABM_EXPIRY = "SET_ABM_EXPIRY",
  SET_APNS_EXPIRY = "SET_APNS_EXPIRY",
  SET_VPP_EXPIRY = "SET_VPP_EXPIRY",
  SET_SANDBOX_EXPIRY = "SET_SANDBOX_EXPIRY",
  SET_NO_SANDBOX_HOSTS = "SET_NO_SANDBOX_HOSTS",
  SET_FILTERED_HOSTS_PATH = "SET_FILTERED_HOSTS_PATH",
  SET_FILTERED_SOFTWARE_PATH = "SET_FILTERED_SOFTWARE_PATH",
  SET_FILTERED_QUERIES_PATH = "SET_FILTERED_QUERIES_PATH",
  SET_FILTERED_POLICIES_PATH = "SET_FILTERED_POLICIES_PATH",
}

interface ISetAvailableTeamsAction {
  type: ACTIONS.SET_AVAILABLE_TEAMS;
  user: IUser | null;
  availableTeams: ITeamSummary[];
}

interface ISetUISettingsAction {
  type: ACTIONS.SET_UI_SETTINGS;
  uiSettings: IUserUISettings;
}

interface ISetConfigAction {
  type: ACTIONS.SET_CONFIG;
  config: IConfig;
}

interface ISetCurrentTeamAction {
  type: ACTIONS.SET_CURRENT_TEAM;
  currentTeam: ITeamSummary | undefined;
}
interface ISetCurrentUserAction {
  type: ACTIONS.SET_CURRENT_USER;
  currentUser: IUser;
}
interface ISetEnrollSecretAction {
  type: ACTIONS.SET_ENROLL_SECRET;
  enrollSecret: IEnrollSecret[];
}

interface IAbmExpiry {
  earliestExpiry: string;
  needsAbmTermsRenewal: boolean;
}

interface ISetABMExpiryAction {
  type: ACTIONS.SET_ABM_EXPIRY;
  abmExpiry: IAbmExpiry;
}

interface ISetAPNsExpiryAction {
  type: ACTIONS.SET_APNS_EXPIRY;
  apnsExpiry: string;
}

interface ISetVppExpiryAction {
  type: ACTIONS.SET_VPP_EXPIRY;
  vppExpiry: string;
}

interface ISetSandboxExpiryAction {
  type: ACTIONS.SET_SANDBOX_EXPIRY;
  sandboxExpiry: string;
}

interface ISetNoSandboxHostsAction {
  type: ACTIONS.SET_NO_SANDBOX_HOSTS;
  noSandboxHosts: boolean;
}

interface ISetFilteredHostsPathAction {
  type: ACTIONS.SET_FILTERED_HOSTS_PATH;
  filteredHostsPath: string;
}

interface ISetFilteredSoftwarePathAction {
  type: ACTIONS.SET_FILTERED_SOFTWARE_PATH;
  filteredSoftwarePath: string;
}

interface ISetFilteredQueriesPathAction {
  type: ACTIONS.SET_FILTERED_QUERIES_PATH;
  filteredQueriesPath: string;
}

interface ISetFilteredPoliciesPathAction {
  type: ACTIONS.SET_FILTERED_POLICIES_PATH;
  filteredPoliciesPath: string;
}
type IAction =
  | ISetAvailableTeamsAction
  | ISetUISettingsAction
  | ISetConfigAction
  | ISetCurrentTeamAction
  | ISetCurrentUserAction
  | ISetEnrollSecretAction
  | ISetABMExpiryAction
  | ISetAPNsExpiryAction
  | ISetVppExpiryAction
  | ISetSandboxExpiryAction
  | ISetNoSandboxHostsAction
  | ISetFilteredHostsPathAction
  | ISetFilteredSoftwarePathAction
  | ISetFilteredQueriesPathAction
  | ISetFilteredPoliciesPathAction;

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  availableTeams?: ITeamSummary[];
  uiSettings?: IUserUISettings;
  config: IConfig | null;
  currentUser: IUser | null;
  currentTeam?: ITeamSummary;
  enrollSecret: IEnrollSecret[] | null;
  isPreviewMode?: boolean;
  isSandboxMode?: boolean;
  isFreeTier?: boolean;
  isPremiumTier?: boolean;
  isMacMdmEnabledAndConfigured?: boolean;
  isWindowsMdmEnabledAndConfigured?: boolean;
  isGlobalAdmin?: boolean;
  isGlobalMaintainer?: boolean;
  isGlobalObserver?: boolean;
  isOnGlobalTeam?: boolean;
  isAnyTeamObserverPlus?: boolean;
  isAnyTeamMaintainer?: boolean;
  isAnyTeamMaintainerOrTeamAdmin?: boolean;
  isTeamObserver?: boolean;
  isTeamMaintainer?: boolean;
  isTeamMaintainerOrTeamAdmin?: boolean;
  isAnyTeamAdmin?: boolean;
  isTeamAdmin?: boolean;
  isOnlyObserver?: boolean;
  isObserverPlus?: boolean;
  isNoAccess?: boolean;
  isAppleBmExpired: boolean;
  isApplePnsExpired: boolean;
  isVppExpired: boolean;
  needsAbmTermsRenewal: boolean;
  willAppleBmExpire: boolean;
  willApplePnsExpire: boolean;
  willVppExpire: boolean;
  abmExpiry?: IAbmExpiry;
  apnsExpiry?: string;
  vppExpiry?: string;
  sandboxExpiry?: string;
  noSandboxHosts?: boolean;
  filteredHostsPath?: string;
  filteredSoftwarePath?: string;
  filteredQueriesPath?: string;
  filteredPoliciesPath?: string;
  isVppEnabled?: boolean;
  setAvailableTeams: (
    user: IUser | null,
    availableTeams: ITeamSummary[]
  ) => void;
  setUISettings: (uiSettings: IUserUISettings) => void;
  setCurrentUser: (user: IUser) => void;
  setCurrentTeam: (team?: ITeamSummary) => void;
  setConfig: (config: IConfig) => void;
  setEnrollSecret: (enrollSecret: IEnrollSecret[]) => void;
  setAPNsExpiry: (apnsExpiry: string) => void;
  setABMExpiry: (abmExpiry: IAbmExpiry) => void;
  setVppExpiry: (vppExpiry: string) => void;
  setSandboxExpiry: (sandboxExpiry: string) => void;
  setNoSandboxHosts: (noSandboxHosts: boolean) => void;
  setFilteredHostsPath: (filteredHostsPath: string) => void;
  setFilteredSoftwarePath: (filteredSoftwarePath: string) => void;
  setFilteredQueriesPath: (filteredQueriesPath: string) => void;
  setFilteredPoliciesPath: (filteredPoliciesPath: string) => void;
};

export type IAppContext = InitialStateType;

export const initialState = {
  availableTeams: undefined,
  uiSettings: undefined,
  config: null,
  currentUser: null,
  currentTeam: undefined,
  enrollSecret: null,
  isPreviewMode: false,
  isSandboxMode: false,
  isFreeTier: undefined,
  isPremiumTier: undefined,
  isMacMdmEnabledAndConfigured: undefined,
  isWindowsMdmEnabledAndConfigured: undefined,
  isGlobalAdmin: undefined,
  isGlobalMaintainer: undefined,
  isGlobalObserver: undefined,
  isOnGlobalTeam: undefined,
  isAnyTeamObserverPlus: undefined,
  isAnyTeamMaintainer: undefined,
  isAnyTeamMaintainerOrTeamAdmin: undefined,
  isTeamObserver: undefined,
  isTeamMaintainer: undefined,
  isTeamMaintainerOrTeamAdmin: undefined,
  isAnyTeamAdmin: undefined,
  isTeamAdmin: undefined,
  isOnlyObserver: undefined,
  isObserverPlus: undefined,
  isNoAccess: undefined,
  filteredHostsPath: undefined,
  filteredSoftwarePath: undefined,
  filteredQueriesPath: undefined,
  filteredPoliciesPath: undefined,
  isAppleBmExpired: false,
  isApplePnsExpired: false,
  isVppExpired: false,
  needsAbmTermsRenewal: false,
  willAppleBmExpire: false,
  willApplePnsExpire: false,
  willVppExpire: false,
  setAvailableTeams: () => null,
  setUISettings: () => null,
  setCurrentUser: () => null,
  setCurrentTeam: () => null,
  setConfig: () => null,
  setEnrollSecret: () => null,
  setAPNsExpiry: () => null,
  setABMExpiry: () => null,
  setVppExpiry: () => null,
  setSandboxExpiry: () => null,
  setNoSandboxHosts: () => null,
  setFilteredHostsPath: () => null,
  setFilteredSoftwarePath: () => null,
  setFilteredQueriesPath: () => null,
  setFilteredPoliciesPath: () => null,
};

const detectPreview = () => {
  return window.location.origin === "http://localhost:1337";
};

// helper function - this is run every
// time currentUser, currentTeam, config, or teamId is changed
const setPermissions = (
  user: IUser | null,
  config: IConfig | null,
  teamId = APP_CONTEXT_NO_TEAM_ID
) => {
  if (!user || !config) {
    return {};
  }

  if (teamId < APP_CONTEXT_NO_TEAM_ID) {
    teamId = APP_CONTEXT_NO_TEAM_ID;
  }

  return {
    isSandboxMode: permissions.isSandboxMode(config),
    isFreeTier: permissions.isFreeTier(config),
    isPremiumTier: permissions.isPremiumTier(config),
    isMacMdmEnabledAndConfigured: permissions.isMacMdmEnabledAndConfigured(
      config
    ),
    isWindowsMdmEnabledAndConfigured: permissions.isWindowsMdmEnabledAndConfigured(
      config
    ),
    isGlobalAdmin: permissions.isGlobalAdmin(user),
    isGlobalMaintainer: permissions.isGlobalMaintainer(user),
    isGlobalObserver: permissions.isGlobalObserver(user),
    isOnGlobalTeam: permissions.isOnGlobalTeam(user),
    isAnyTeamObserverPlus: permissions.isAnyTeamObserverPlus(user),
    isAnyTeamMaintainer: permissions.isAnyTeamMaintainer(user),
    isAnyTeamMaintainerOrTeamAdmin: permissions.isAnyTeamMaintainerOrTeamAdmin(
      user
    ),
    isAnyTeamAdmin: permissions.isAnyTeamAdmin(user),
    isTeamObserver: permissions.isTeamObserver(user, teamId),
    isTeamMaintainer: permissions.isTeamMaintainer(user, teamId),
    isTeamAdmin: permissions.isTeamAdmin(user, teamId),
    isTeamMaintainerOrTeamAdmin: permissions.isTeamMaintainerOrTeamAdmin(
      user,
      teamId
    ),
    isOnlyObserver: permissions.isOnlyObserver(user),
    isObserverPlus: permissions.isObserverPlus(user, teamId),
    isNoAccess: permissions.isNoAccess(user),
  };
};

const reducer = (state: InitialStateType, action: IAction) => {
  switch (action.type) {
    case ACTIONS.SET_UI_SETTINGS: {
      const { uiSettings } = action;
      return {
        ...state,
        uiSettings,
      };
    }
    case ACTIONS.SET_AVAILABLE_TEAMS: {
      const { user, availableTeams } = action;

      let sortedTeams = availableTeams.sort(
        (a: ITeamSummary, b: ITeamSummary) =>
          sort.caseInsensitiveAsc(a.name, b.name)
      );
      sortedTeams = sortedTeams.filter(
        (t) =>
          t.name !== APP_CONTEXT_ALL_TEAMS_SUMMARY.name &&
          t.name !== APP_CONTEX_NO_TEAM_SUMMARY.name
      );
      if (user && permissions.isOnGlobalTeam(user)) {
        sortedTeams.unshift(
          APP_CONTEXT_ALL_TEAMS_SUMMARY,
          APP_CONTEX_NO_TEAM_SUMMARY
        );
      }

      return {
        ...state,
        availableTeams: sortedTeams,
      };
    }
    case ACTIONS.SET_CURRENT_USER: {
      const { currentUser } = action;

      return {
        ...state,
        currentUser,
        ...setPermissions(currentUser, state.config, state.currentTeam?.id),
      };
    }
    case ACTIONS.SET_CURRENT_TEAM: {
      const { currentTeam } = action;
      return {
        ...state,
        currentTeam,
        ...setPermissions(state.currentUser, state.config, currentTeam?.id),
      };
    }
    case ACTIONS.SET_CONFIG: {
      const { config } = action;
      return {
        ...state,
        config,
        ...setPermissions(state.currentUser, config, state.currentTeam?.id),
      };
    }
    case ACTIONS.SET_ENROLL_SECRET: {
      const { enrollSecret } = action;
      return {
        ...state,
        enrollSecret,
      };
    }
    case ACTIONS.SET_ABM_EXPIRY: {
      const { abmExpiry } = action;
      const { earliestExpiry, needsAbmTermsRenewal } = abmExpiry;
      return {
        ...state,
        abmExpiry,
        isAppleBmExpired: hasLicenseExpired(earliestExpiry),
        willAppleBmExpire: willExpireWithinXDays(earliestExpiry, 30),
        needsAbmTermsRenewal,
      };
    }
    case ACTIONS.SET_APNS_EXPIRY: {
      const { apnsExpiry } = action;
      return {
        ...state,
        apnsExpiry,
        isApplePnsExpired: hasLicenseExpired(apnsExpiry),
        willApplePnsExpire: willExpireWithinXDays(apnsExpiry, 30),
      };
    }
    case ACTIONS.SET_VPP_EXPIRY: {
      const { vppExpiry } = action;
      return {
        ...state,
        vppExpiry,
        isVppExpired: hasLicenseExpired(vppExpiry),
        willVppExpire: willExpireWithinXDays(vppExpiry, 30),
      };
    }
    case ACTIONS.SET_SANDBOX_EXPIRY: {
      const { sandboxExpiry } = action;
      return {
        ...state,
        sandboxExpiry,
      };
    }
    case ACTIONS.SET_NO_SANDBOX_HOSTS: {
      const { noSandboxHosts } = action;
      return {
        ...state,
        noSandboxHosts,
      };
    }
    case ACTIONS.SET_FILTERED_HOSTS_PATH: {
      const { filteredHostsPath } = action;
      return {
        ...state,
        filteredHostsPath,
      };
    }
    case ACTIONS.SET_FILTERED_SOFTWARE_PATH: {
      const { filteredSoftwarePath } = action;
      return {
        ...state,
        filteredSoftwarePath,
      };
    }
    case ACTIONS.SET_FILTERED_QUERIES_PATH: {
      const { filteredQueriesPath } = action;
      return {
        ...state,
        filteredQueriesPath,
      };
    }
    case ACTIONS.SET_FILTERED_POLICIES_PATH: {
      const { filteredPoliciesPath } = action;
      return {
        ...state,
        filteredPoliciesPath,
      };
    }
    default:
      return state;
  }
};

export const AppContext = createContext<InitialStateType>(initialState);

const AppProvider = ({ children }: Props): JSX.Element => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = useMemo(
    () => ({
      availableTeams: state.availableTeams,
      uiSettings: state.uiSettings,
      config: state.config,
      currentUser: state.currentUser,
      currentTeam: state.currentTeam,
      enrollSecret: state.enrollSecret,
      sandboxExpiry: state.sandboxExpiry,
      abmExpiry: state.abmExpiry,
      apnsExpiry: state.apnsExpiry,
      vppExpiry: state.vppExpiry,
      isAppleBmExpired: state.isAppleBmExpired,
      isApplePnsExpired: state.isApplePnsExpired,
      isVppExpired: state.isVppExpired,
      needsAbmTermsRenewal: state.needsAbmTermsRenewal,
      willAppleBmExpire: state.willAppleBmExpire,
      willApplePnsExpire: state.willApplePnsExpire,
      willVppExpire: state.willVppExpire,
      noSandboxHosts: state.noSandboxHosts,
      filteredHostsPath: state.filteredHostsPath,
      filteredSoftwarePath: state.filteredSoftwarePath,
      filteredQueriesPath: state.filteredQueriesPath,
      filteredPoliciesPath: state.filteredPoliciesPath,
      isPreviewMode: detectPreview(),
      isSandboxMode: state.isSandboxMode,
      isFreeTier: state.isFreeTier,
      isPremiumTier: state.isPremiumTier,
      isMacMdmEnabledAndConfigured: state.isMacMdmEnabledAndConfigured,
      isWindowsMdmEnabledAndConfigured: state.isWindowsMdmEnabledAndConfigured,
      isGlobalAdmin: state.isGlobalAdmin,
      isGlobalMaintainer: state.isGlobalMaintainer,
      isGlobalObserver: state.isGlobalObserver,
      isOnGlobalTeam: state.isOnGlobalTeam,
      isAnyTeamObserverPlus: state.isAnyTeamObserverPlus,
      isAnyTeamMaintainer: state.isAnyTeamMaintainer,
      isAnyTeamMaintainerOrTeamAdmin: state.isAnyTeamMaintainerOrTeamAdmin,
      isTeamObserver: state.isTeamObserver,
      isTeamMaintainer: state.isTeamMaintainer,
      isTeamAdmin: state.isTeamAdmin,
      isTeamMaintainerOrTeamAdmin: state.isTeamMaintainerOrTeamAdmin,
      isAnyTeamAdmin: state.isAnyTeamAdmin,
      isOnlyObserver: state.isOnlyObserver,
      isObserverPlus: state.isObserverPlus,
      isNoAccess: state.isNoAccess,
      setAvailableTeams: (
        user: IUser | null,
        availableTeams: ITeamSummary[]
      ) => {
        dispatch({
          type: ACTIONS.SET_AVAILABLE_TEAMS,
          user,
          availableTeams,
        });
      },
      setUISettings: (uiSettings: IUserUISettings) => {
        dispatch({ type: ACTIONS.SET_UI_SETTINGS, uiSettings });
      },
      setCurrentUser: (currentUser: IUser) => {
        dispatch({ type: ACTIONS.SET_CURRENT_USER, currentUser });
      },
      setCurrentTeam: (currentTeam: ITeamSummary | undefined) => {
        dispatch({ type: ACTIONS.SET_CURRENT_TEAM, currentTeam });
      },
      setConfig: (config: IConfig) => {
        dispatch({ type: ACTIONS.SET_CONFIG, config });
      },
      setEnrollSecret: (enrollSecret: IEnrollSecret[]) => {
        dispatch({ type: ACTIONS.SET_ENROLL_SECRET, enrollSecret });
      },
      setABMExpiry: (abmExpiry: IAbmExpiry) => {
        dispatch({ type: ACTIONS.SET_ABM_EXPIRY, abmExpiry });
      },
      setAPNsExpiry: (apnsExpiry: string) => {
        dispatch({ type: ACTIONS.SET_APNS_EXPIRY, apnsExpiry });
      },
      setVppExpiry: (vppExpiry: string) => {
        dispatch({
          type: ACTIONS.SET_VPP_EXPIRY,
          vppExpiry,
        });
      },
      setSandboxExpiry: (sandboxExpiry: string) => {
        dispatch({ type: ACTIONS.SET_SANDBOX_EXPIRY, sandboxExpiry });
      },
      setNoSandboxHosts: (noSandboxHosts: boolean) => {
        dispatch({
          type: ACTIONS.SET_NO_SANDBOX_HOSTS,
          noSandboxHosts,
        });
      },
      setFilteredHostsPath: (filteredHostsPath: string) => {
        dispatch({ type: ACTIONS.SET_FILTERED_HOSTS_PATH, filteredHostsPath });
      },
      setFilteredSoftwarePath: (filteredSoftwarePath: string) => {
        dispatch({
          type: ACTIONS.SET_FILTERED_SOFTWARE_PATH,
          filteredSoftwarePath,
        });
      },
      setFilteredQueriesPath: (filteredQueriesPath: string) => {
        dispatch({
          type: ACTIONS.SET_FILTERED_QUERIES_PATH,
          filteredQueriesPath,
        });
      },
      setFilteredPoliciesPath: (filteredPoliciesPath: string) => {
        dispatch({
          type: ACTIONS.SET_FILTERED_POLICIES_PATH,
          filteredPoliciesPath,
        });
      },
    }),
    [
      state.abmExpiry,
      state.apnsExpiry,
      state.availableTeams,
      state.uiSettings,
      state.config,
      state.currentTeam,
      state.currentUser,
      state.enrollSecret,
      state.filteredHostsPath,
      state.filteredPoliciesPath,
      state.filteredQueriesPath,
      state.filteredSoftwarePath,
      state.isAnyTeamAdmin,
      state.isAnyTeamMaintainer,
      state.isAnyTeamMaintainerOrTeamAdmin,
      state.isAnyTeamObserverPlus,
      state.isAppleBmExpired,
      state.isApplePnsExpired,
      state.isFreeTier,
      state.isGlobalAdmin,
      state.isGlobalMaintainer,
      state.isGlobalObserver,
      state.isMacMdmEnabledAndConfigured,
      state.isNoAccess,
      state.isObserverPlus,
      state.isOnGlobalTeam,
      state.isOnlyObserver,
      state.isPremiumTier,
      state.isSandboxMode,
      state.isTeamAdmin,
      state.isTeamMaintainer,
      state.isTeamMaintainerOrTeamAdmin,
      state.isTeamObserver,
      state.isVppExpired,
      state.isWindowsMdmEnabledAndConfigured,
      state.needsAbmTermsRenewal,
      state.noSandboxHosts,
      state.sandboxExpiry,
      state.vppExpiry,
      state.willAppleBmExpire,
      state.willApplePnsExpire,
      state.willVppExpire,
    ]
  );
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppProvider;
