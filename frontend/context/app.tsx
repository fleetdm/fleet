import React, { createContext, useReducer, ReactNode } from "react";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import { ITeam } from "interfaces/team";
import permissions from "utilities/permissions";
import { IEnrollSecret } from "interfaces/enroll_secret";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  config: IConfig | null;
  currentUser: IUser | null;
  currentTeam: ITeam | undefined;
  enrollSecret: IEnrollSecret[] | null;
  isFreeTier: boolean | undefined;
  isPremiumTier: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  isGlobalMaintainer: boolean | undefined;
  isGlobalObserver: boolean | undefined;
  isOnGlobalTeam: boolean | undefined;
  isAnyTeamMaintainer: boolean | undefined;
  isTeamMaintainer: boolean | undefined;
  isOnlyObserver: boolean | undefined;
  setCurrentUser: (user: IUser) => void;
  setCurrentTeam: (team: ITeam | undefined) => void;
  setConfig: (config: IConfig) => void;
  setEnrollSecret: (enrollSecret: IEnrollSecret[]) => void;
};

const initialState = {
  config: null,
  currentUser: null,
  currentTeam: undefined,
  enrollSecret: null,
  isFreeTier: undefined,
  isPremiumTier: undefined,
  isGlobalAdmin: undefined,
  isGlobalMaintainer: undefined,
  isGlobalObserver: undefined,
  isOnGlobalTeam: undefined,
  isAnyTeamMaintainer: undefined,
  isTeamMaintainer: undefined,
  isOnlyObserver: undefined,
  setCurrentUser: () => null,
  setCurrentTeam: () => null,
  setConfig: () => null,
  setEnrollSecret: () => null,
};

const actions = {
  SET_CURRENT_USER: "SET_CURRENT_USER",
  SET_CURRENT_TEAM: "SET_CURRENT_TEAM",
  SET_CONFIG: "SET_CONFIG",
  SET_ENROLL_SECRET: "SET_ENROLL_SECRET",
};

// helper function - this is run every
// time currentUser, config, or teamId is changed
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
    isTeamMaintainer: permissions.isTeamMaintainer(user, teamId),
    isOnlyObserver: permissions.isOnlyObserver(user),
  };
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_CURRENT_USER:
      return {
        ...state,
        currentUser: action.currentUser,
        ...setPermissions(action.currentUser, state.config),
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

const AppProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    config: state.config,
    currentUser: state.currentUser,
    currentTeam: state.currentTeam,
    enrollSecret: state.enrollSecret,
    isFreeTier: state.isFreeTier,
    isPremiumTier: state.isPremiumTier,
    isGlobalAdmin: state.isGlobalAdmin,
    isGlobalMaintainer: state.isGlobalMaintainer,
    isGlobalObserver: state.isGlobalObserver,
    isOnGlobalTeam: state.isOnGlobalTeam,
    isAnyTeamMaintainer: state.isAnyTeamMaintainer,
    isTeamMaintainer: state.isTeamMaintainer,
    isOnlyObserver: state.isOnlyObserver,
    setCurrentUser: (currentUser: IUser) => {
      dispatch({ type: actions.SET_CURRENT_USER, currentUser });
    },
    setCurrentTeam: (currentTeam: ITeam | undefined) => {
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
