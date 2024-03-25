import React, { createContext, useReducer, ReactNode } from "react";

import { IConfig } from "interfaces/config";
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

enum ACTIONS {
  SET_AVAILABLE_TEAMS = "SET_AVAILABLE_TEAMS",
  SET_CURRENT_USER = "SET_CURRENT_USER",
  SET_CURRENT_TEAM = "SET_CURRENT_TEAM",
  SET_CONFIG = "SET_CONFIG",
  SET_ENROLL_SECRET = "SET_ENROLL_SECRET",
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
  | ISetConfigAction
  | ISetCurrentTeamAction
  | ISetCurrentUserAction
  | ISetEnrollSecretAction
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
  sandboxExpiry?: string;
  noSandboxHosts?: boolean;
  filteredHostsPath?: string;
  filteredSoftwarePath?: string;
  filteredQueriesPath?: string;
  filteredPoliciesPath?: string;
  setAvailableTeams: (
    user: IUser | null,
    availableTeams: ITeamSummary[]
  ) => void;
  setCurrentUser: (user: IUser) => void;
  setCurrentTeam: (team?: ITeamSummary) => void;
  setConfig: (config: IConfig) => void;
  setEnrollSecret: (enrollSecret: IEnrollSecret[]) => void;
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
  setAvailableTeams: () => null,
  setCurrentUser: () => null,
  setCurrentTeam: () => null,
  setConfig: () => null,
  setEnrollSecret: () => null,
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
      // config.sandbox_enabled = true; // TODO: uncomment for sandbox dev

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
      // TODO: if policies page is updated to support team_id=0, remove the replace below
      return {
        ...state,
        filteredPoliciesPath: filteredPoliciesPath.replace("team_id=0", ""),
      };
    }
    default:
      return state;
  }
};

export const AppContext = createContext<InitialStateType>(initialState);

const AppProvider = ({ children }: Props): JSX.Element => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    availableTeams: state.availableTeams,
    config: state.config,
    currentUser: state.currentUser,
    currentTeam: state.currentTeam,
    enrollSecret: state.enrollSecret,
    sandboxExpiry: state.sandboxExpiry,
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
    setAvailableTeams: (user: IUser | null, availableTeams: ITeamSummary[]) => {
      dispatch({
        type: ACTIONS.SET_AVAILABLE_TEAMS,
        user,
        availableTeams,
      });
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
  };
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppProvider;
