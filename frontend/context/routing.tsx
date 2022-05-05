import React, { createContext, useReducer, ReactNode } from "react";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  redirectLocation: string | null;
  setRedirectLocation: (pathname: string | null) => void;
};

const initialState = {
  redirectLocation: null,
  setRedirectLocation: () => null,
};

const actions = {
  SET_REDIRECT_LOCATION: "SET_REDIRECT_LOCATION",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_REDIRECT_LOCATION:
      return { ...state, redirectLocation: action.pathname };
    default:
      return state;
  }
};

export const RoutingContext = createContext<InitialStateType>(initialState);

const RoutingProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    redirectLocation: state.redirectLocation,
    setRedirectLocation: (pathname: string | null) => {
      dispatch({ type: actions.SET_REDIRECT_LOCATION, pathname });
    },
  };

  return (
    <RoutingContext.Provider value={value}>{children}</RoutingContext.Provider>
  );
};

export default RoutingProvider;
