import React, { createContext, useReducer, ReactNode } from "react";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import permissions from "utilities/permissions";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  currentUser: IUser | null;
  config: IConfig | null;
  isFreeTier: boolean | undefined;
  isPremiumTier: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  isGlobalMaintainer: boolean | undefined;
  isGlobalObserver: boolean | undefined;
  isOnGlobalTeam: boolean | undefined;
  isAnyTeamMaintainer: boolean | undefined;
  isOnlyObserver: boolean | undefined;
  setCurrentUser: (user: IUser) => void;
  setConfig: (config: IConfig) => void;
};

const initialState = {
  currentUser: null,
  config: null,
  isFreeTier: undefined,
  isPremiumTier: undefined,
  isGlobalAdmin: undefined,
  isGlobalMaintainer: undefined,
  isGlobalObserver: undefined,
  isOnGlobalTeam: undefined,
  isAnyTeamMaintainer: undefined,
  isOnlyObserver: undefined,
  setCurrentUser: () => null,
  setConfig: () => null,
};

const actions = {
  SET_CURRENT_USER: "SET_CURRENT_USER",
  SET_CONFIG: "SET_CONFIG",
};

// helper function - this is run every
// time currentUser or config is changed
const setPermissions = (user: IUser, config: IConfig) => {
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
    case actions.SET_CONFIG:
      return {
        ...state,
        config: action.config,
        ...setPermissions(state.currentUser, action.config),
      };
    default:
      return state;
  }
};

export const AppContext = createContext<InitialStateType>(initialState);

const AppProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    currentUser: state.currentUser,
    config: state.config,
    isFreeTier: state.isFreeTier,
    isPremiumTier: state.isPremiumTier,
    isGlobalAdmin: state.isGlobalAdmin,
    isGlobalMaintainer: state.isGlobalMaintainer,
    isGlobalObserver: state.isGlobalObserver,
    isOnGlobalTeam: state.isOnGlobalTeam,
    isAnyTeamMaintainer: state.isAnyTeamMaintainer,
    isOnlyObserver: state.isOnlyObserver,
    setCurrentUser: (currentUser: IUser) => {
      dispatch({ type: actions.SET_CURRENT_USER, currentUser });
    },
    setConfig: (config: IConfig) => {
      dispatch({ type: actions.SET_CONFIG, config });
    },
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppProvider;
