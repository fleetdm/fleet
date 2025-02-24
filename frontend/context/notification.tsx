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

type FlashOptions = {
  /** `persistOnPageChange` is used to keep the flash message showing after a
   * router change if set to `true`.
   *
   * @default undefined
   * */
  persistOnPageChange?: boolean;
};

type MultiFlashOptions = FlashOptions & {
  notifications?: INotification[];
};

type InitialStateType = {
  notification: INotification | null;
  renderFlash: (
    alertType: "success" | "error" | "warning-filled" | null,
    message: JSX.Element | string | null,
    options?: FlashOptions
  ) => void;
  renderMultiFlash: (options?: MultiFlashOptions) => void;
  hideFlash: (id?: string) => void;
};

export type INotificationContext = InitialStateType;

const initialState = {
  notification: null,
  renderFlash: noop,
  renderMultiFlash: noop,
  hideFlash: noop,
};

const actionTypes = {
  RENDER_FLASH: "RENDER_FLASH",
  RENDER_MULTIFLASH: "RENDER_MULTIFLASH",
  HIDE_FLASH: "HIDE_FLASH",
} as const;

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actionTypes.RENDER_MULTIFLASH: {
      const newNotifications = action.notifications;

      const updatedNotifications = state.notification.concat(newNotifications);

      return {
        ...state,
        notification: updatedNotifications,
      };
    }
    case actionTypes.RENDER_FLASH: {
      return {
        ...state,
        notification: {
          alertType: action.alertType,
          isVisible: true,
          message: action.message,
          persistOnPageChange: action.options?.persistOnPageChange ?? false,
        },
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
      }
    ) => {
      setTimeout(() => {
        const newNotification = {
          alertType,
          isVisible: true,
          message,
          persistOnPageChange: options?.persistOnPageChange ?? false,
        };
        dispatch({
          type: actionTypes.RENDER_FLASH,
          notifications: [newNotification],
        });
      });
    },
    []
  );

  const renderMultiFlash = useCallback((options?: MultiFlashOptions) => {
    setTimeout(() => {
      if (options?.notifications) {
        dispatch({
          type: actionTypes.RENDER_MULTIFLASH,
          notifications: options.notifications,
        });
      }
    });
  }, []);

  const hideFlash = useCallback((id?: string) => {
    dispatch({ type: actionTypes.HIDE_FLASH, id });
  }, []);

  const value = useMemo(
    () => ({
      notification: state.notification,
      renderFlash,
      renderMultiFlash,
      hideFlash,
    }),
    [state.notification, renderFlash, renderMultiFlash, hideFlash]
  );

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export default NotificationProvider;
