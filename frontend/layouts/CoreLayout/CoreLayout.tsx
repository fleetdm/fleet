import React, { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";

import UnsupportedScreenSize from "layouts/UnsupportedScreenSize";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { TableContext } from "context/table";

import paths from "router/paths";
import useDeepEffect from "hooks/useDeepEffect";
import FlashMessage from "components/FlashMessage";
import SiteTopNav from "components/top_nav/SiteTopNav";
import { QueryParams } from "utilities/url";

interface ICoreLayoutProps {
  children: React.ReactNode;
  router: InjectedRouter; // v3
  // TODO: standardize typing and usage of location across app components
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
}

const CoreLayout = ({ children, router, location }: ICoreLayoutProps) => {
  const { config, currentUser } = useContext(AppContext);
  const { notification, hideFlash } = useContext(NotificationContext);
  const { setResetSelectedRows } = useContext(TableContext);

  // on success of an action, the table will reset its checkboxes.
  // setTimeout is to help with race conditions as table reloads
  // in some instances (i.e. Manage Hosts)
  useDeepEffect(() => {
    if (notification?.alertType === "success") {
      setTimeout(() => {
        setResetSelectedRows(true);
        setTimeout(() => {
          setResetSelectedRows(false);
        }, 300);
      }, 0);
    }
  }, [notification]);

  const onLogoutUser = async () => {
    const { LOGOUT } = paths;
    router.push(LOGOUT);
  };

  const onUserMenuItemClick = (path: string) => {
    return (evt: React.MouseEvent<HTMLButtonElement>) => {
      evt.preventDefault();

      if (path.indexOf("http") !== -1) {
        global.window.open(path, "_blank");
        return false;
      }

      router.push(path);
      return false;
    };
  };

  const onUndoActionClick = (undoAction?: () => void) => {
    return (evt: React.MouseEvent<HTMLButtonElement>) => {
      evt.preventDefault();

      if (undoAction) {
        undoAction();
      }

      hideFlash();
    };
  };

  const fullWidthFlash = !currentUser;

  if (!currentUser || !config) {
    return null;
  }

  return (
    <div className="app-wrap">
      <UnsupportedScreenSize />
      <nav className="site-nav-container">
        <SiteTopNav
          config={config}
          currentUser={currentUser}
          location={location}
          onLogoutUser={onLogoutUser}
          onUserMenuItemClick={onUserMenuItemClick}
        />
      </nav>
      <div className="core-wrapper">
        <FlashMessage
          fullWidth={fullWidthFlash}
          notification={notification}
          onRemoveFlash={hideFlash}
          onUndoActionClick={onUndoActionClick}
          pathname={location.pathname}
        />

        {children}
      </div>
    </div>
  );
};

export default CoreLayout;
