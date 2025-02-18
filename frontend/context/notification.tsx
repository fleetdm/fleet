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
    options?: { persistOnPageChange?: boolean }
  ) => void;
  hideFlash: () => void;
};

export type INotificationContext = InitialStateType;

const initialState = {
  notification: null,
  renderFlash: noop,
  hideFlash: noop,
};

const actionTypes = {
  RENDER_FLASH: "RENDER_FLASH",
  HIDE_FLASH: "HIDE_FLASH",
} as const;

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actionTypes.RENDER_FLASH:
      return {
        ...state,
        notification: {
          alertType: action.alertType,
          isVisible: true,
          message: action.message,
          persistOnPageChange: action.options?.persistOnPageChange ?? false,
        },
      };
    case actionTypes.HIDE_FLASH:
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
      options?: {
        persistOnPageChange?: boolean;
      }
    ) => {
      // wrapping the dispatch in a timeout ensures it is evaluated on the next event loop,
      // preventing bugs related to the FlashMessage's self-hiding behavior on URL changes.
      // react router v3 router.push is asynchronous
      setTimeout(() => {
        dispatch({
          type: actionTypes.RENDER_FLASH,
          alertType,
          message,
          options,
        });
      });
    },
    []
  );

  const hideFlash = useCallback(() => {
    dispatch({ type: actionTypes.HIDE_FLASH });
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
