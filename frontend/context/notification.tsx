import React, {
  createContext,
  useReducer,
  ReactNode,
  useCallback,
  useMemo,
} from "react";
import { INotification } from "interfaces/notification";
import { noop } from "lodash";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  notification: INotification | null;
  renderFlash: (
    alertType: "success" | "error" | "warning-filled" | null,
    message: JSX.Element | string | null,
    undoAction?: (evt: React.MouseEvent<HTMLButtonElement>) => void
  ) => void;
  hideFlash: () => void;
};

export type INotificationContext = InitialStateType;

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

  const renderFlash = useCallback(
    (
      alertType: "success" | "error" | "warning-filled" | null,
      message: JSX.Element | string | null,
      undoAction?: (evt: React.MouseEvent<HTMLButtonElement>) => void
    ) => {
      dispatch({
        type: actions.RENDER_FLASH,
        alertType,
        message,
        undoAction,
      });
    },
    []
  );

  const hideFlash = useCallback(() => {
    dispatch({ type: actions.HIDE_FLASH });
  }, []);

  const value = useMemo(
    () => ({
      notification: state.notification,
      renderFlash,
      hideFlash,
    }),
    [state.notification, renderFlash, hideFlash]
  );

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export default NotificationProvider;
