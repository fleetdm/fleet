import React, { createContext, useReducer, ReactNode } from "react";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import { ITeamSummary } from "interfaces/team";
import permissions from "utilities/permissions";
import { IEnrollSecret } from "interfaces/enroll_secret";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  availableTeams: ITeamSummary[] | undefined;
  config: IConfig | null;
  currentUser: IUser | null;
  currentTeam: ITeamSummary | undefined;
  enrollSecret: IEnrollSecret[] | null;
  isPreviewMode: boolean | undefined;
  isFreeTier: boolean | undefined;
  isPremiumTier: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  isGlobalMaintainer: boolean | undefined;
  isGlobalObserver: boolean | undefined;
  isOnGlobalTeam: boolean | undefined;
  isAnyTeamMaintainer: boolean | undefined;
  isAnyTeamMaintainerOrTeamAdmin: boolean | undefined;
  isTeamObserver: boolean | undefined;
  isTeamMaintainer: boolean | undefined;
  isTeamMaintainerOrTeamAdmin: boolean | undefined;
  isAnyTeamAdmin: boolean | undefined;
  isTeamAdmin: boolean | undefined;
  isOnlyObserver: boolean | undefined;
  isNoAccess: boolean | undefined;
  setAvailableTeams: (availableTeams: ITeamSummary[]) => void;
  setCurrentUser: (user: IUser) => void;
  setCurrentTeam: (team: ITeamSummary | undefined) => void;
  setConfig: (config: IConfig) => void;
  setEnrollSecret: (enrollSecret: IEnrollSecret[]) => void;
};

export type IAppContext = InitialStateType;

const initialState = {
  availableTeams: undefined,
  config: null,
  currentUser: null,
  currentTeam: undefined,
  enrollSecret: null,
  isPreviewMode: false,
  isFreeTier: undefined,
  isPremiumTier: undefined,
  isGlobalAdmin: undefined,
  isGlobalMaintainer: undefined,
  isGlobalObserver: undefined,
  isOnGlobalTeam: undefined,
  isAnyTeamMaintainer: undefined,
  isAnyTeamMaintainerOrTeamAdmin: undefined,
  isTeamObserver: undefined,
  isTeamMaintainer: undefined,
  isTeamMaintainerOrTeamAdmin: undefined,
  isAnyTeamAdmin: undefined,
  isTeamAdmin: undefined,
  isOnlyObserver: undefined,
  isNoAccess: undefined,
  setAvailableTeams: () => null,
  setCurrentUser: () => null,
  setCurrentTeam: () => null,
  setConfig: () => null,
  setEnrollSecret: () => null,
};

const actions = {
  SET_AVAILABLE_TEAMS: "SET_AVAILABLE_TEAMS",
  SET_CURRENT_USER: "SET_CURRENT_USER",
  SET_CURRENT_TEAM: "SET_CURRENT_TEAM",
  SET_CONFIG: "SET_CONFIG",
  SET_ENROLL_SECRET: "SET_ENROLL_SECRET",
};

const detectPreview = () => {
  return window.location.origin === "http://localhost:1337";
};

// helper function - this is run every
// time currentUser, currentTeam, config, or teamId is changed
const setPermissions = (user: IUser, config: IConfig, teamId = 0) => {
  if (!user || !config) {
    return {};
  }

  return {
    isFreeTier: permissions.isFreeTier(config),
    isPremiumTier: permissions.isPremiumTier(config),
    isGlobalAdmin: permissions.isGlobalAdmin(user),
    isGlobalMaintainer: permissions.isGlobalMaintainer(user),
    isGlobalObserver: permissions.isGlobalObserver(user),
    isOnGlobalTeam: permissions.isOnGlobalTeam(user),
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
    isNoAccess: permissions.isNoAccess(user),
  };
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_AVAILABLE_TEAMS:
      return {
        ...state,
        availableTeams: action.availableTeams,
      };
    case actions.SET_CURRENT_USER:
      return {
        ...state,
        currentUser: action.currentUser,
        ...setPermissions(
          action.currentUser,
          state.config,
          state.currentTeam?.id
        ),
      };
    case actions.SET_CURRENT_TEAM:
      return {
        ...state,
        currentTeam: action.currentTeam,
        ...setPermissions(
          state.currentUser,
          state.config,
          action.currentTeam?.id
        ),
      };
    case actions.SET_CONFIG:
      return {
        ...state,
        config: action.config,
        ...setPermissions(state.currentUser, action.config),
      };
    case actions.SET_ENROLL_SECRET:
      return {
        ...state,
        enrollSecret: action.enrollSecret,
      };
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
    isPreviewMode: detectPreview(),
    isFreeTier: state.isFreeTier,
    isPremiumTier: state.isPremiumTier,
    isGlobalAdmin: state.isGlobalAdmin,
    isGlobalMaintainer: state.isGlobalMaintainer,
    isGlobalObserver: state.isGlobalObserver,
    isOnGlobalTeam: state.isOnGlobalTeam,
    isAnyTeamMaintainer: state.isAnyTeamMaintainer,
    isAnyTeamMaintainerOrTeamAdmin: state.isAnyTeamMaintainerOrTeamAdmin,
    isTeamObserver: state.isTeamObserver,
    isTeamMaintainer: state.isTeamMaintainer,
    isTeamAdmin: state.isTeamAdmin,
    isTeamMaintainerOrTeamAdmin: state.isTeamMaintainer,
    isAnyTeamAdmin: state.isAnyTeamAdmin,
    isOnlyObserver: state.isOnlyObserver,
    isNoAccess: state.isNoAccess,
    setAvailableTeams: (availableTeams: ITeamSummary[]) => {
      dispatch({ type: actions.SET_AVAILABLE_TEAMS, availableTeams });
    },
    setCurrentUser: (currentUser: IUser) => {
      dispatch({ type: actions.SET_CURRENT_USER, currentUser });
    },
    setCurrentTeam: (currentTeam: ITeamSummary | undefined) => {
      dispatch({ type: actions.SET_CURRENT_TEAM, currentTeam });
    },
    setConfig: (config: IConfig) => {
      dispatch({ type: actions.SET_CONFIG, config });
    },
    setEnrollSecret: (enrollSecret: IEnrollSecret[]) => {
      dispatch({ type: actions.SET_ENROLL_SECRET, enrollSecret });
    },
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppProvider;
