import React, { createContext, useReducer, ReactNode } from "react";
import { INotification } from "interfaces/notification";
import { noop } from "lodash";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  notification: INotification | null;
  renderFlash: (
    alertType: "success" | "error" | "warning-filled" | null,
    message: string | null,
    undoAction?: (evt: React.MouseEvent<HTMLButtonElement>) => void
  ) => void;
  hideFlash: () => void;
};

const initialState = {
  notification: null,
  renderFlash: noop,
  hideFlash: noop,
};

const actions = {
  RENDER_FLASH: "RENDER_FLASH",
  HIDE_FLASH: "HIDE_FLASH",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.RENDER_FLASH:
      return {
        ...state,
        notification: {
          alertType: action.alertType,
          isVisible: true,
          message: action.message,
          undoAction: action.undoAction,
        },
      };
    case actions.HIDE_FLASH:
      return initialState;
    default:
      return state;
  }
};

export const NotificationContext = createContext<InitialStateType>(
  initialState
);

const NotificationProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    notification: state.notification,
    renderFlash: (
      alertType: "success" | "error" | "warning-filled" | null,
      message: string | null,
      undoAction?: (evt: React.MouseEvent<HTMLButtonElement>) => void
    ) => {
      dispatch({
        type: actions.RENDER_FLASH,
        alertType,
        message,
        undoAction,
      });
    },
    hideFlash: () => {
      dispatch({ type: actions.HIDE_FLASH });
    },
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export default NotificationProvider;
