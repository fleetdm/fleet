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
  notification: INotification | INotification[] | null;
  renderFlash: (
    alertType: "success" | "error" | "warning-filled" | null,
    message: JSX.Element | string | null,
    options?: {
      /** `persistOnPageChange` is used to keep the flash message showing after a
       * router change if set to `true`.
       *
       * @default undefined
       * */
      persistOnPageChange?: boolean;
      id?: string;
      notifications?: INotification[];
    }
  ) => void;
  hideFlash: (id?: string) => void;
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
    case actionTypes.RENDER_FLASH: {
      let newNotifications;

      if (Array.isArray(action.notifications)) {
        // If we receive an array of notifications, use it directly
        newNotifications = action.notifications;
      } else {
        // Otherwise, create a single notification object
        newNotifications = [
          {
            id: action.id || Date.now().toString(),
            alertType: action.alertType,
            isVisible: true,
            message: action.message,
            persistOnPageChange: action.options?.persistOnPageChange ?? false,
          },
        ];
      }

      // If the current state is an array, concatenate; otherwise, use the new notifications
      const updatedNotifications = Array.isArray(state.notification)
        ? state.notification.concat(newNotifications)
        : newNotifications;

      return {
        ...state,
        notification: updatedNotifications,
      };
    }
    case actionTypes.HIDE_FLASH:
      if (Array.isArray(state.notification)) {
        return {
          ...state,
          notification: state.notification.filter(
            (n: INotification) => n.id !== action.id
          ),
        };
      }
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
        id?: string;
        notifications?: INotification[];
      }
    ) => {
      setTimeout(() => {
        if (options?.notifications) {
          dispatch({
            type: actionTypes.RENDER_FLASH,
            notifications: options.notifications,
          });
        } else {
          dispatch({
            type: actionTypes.RENDER_FLASH,
            id: options?.id || Date.now().toString(),
            alertType,
            message,
            options,
          });
        }
      });
    },
    []
  );

  const hideFlash = useCallback((id?: string) => {
    dispatch({ type: actionTypes.HIDE_FLASH, id });
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
