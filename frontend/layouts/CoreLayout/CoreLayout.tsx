import React, { useState, useContext } from "react";
import { InjectedRouter } from "react-router";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { TableContext } from "context/table";

import paths from "router/paths";
import useDeepEffect from "hooks/useDeepEffect";
import FlashMessage from "components/FlashMessage";
import SiteTopNav from "components/top_nav/SiteTopNav";
import { INotification } from "interfaces/notification";
import { licenseExpirationWarning } from "utilities/helpers";
import ExternalLinkIcon from "../../../assets/images/icon-external-link-12x12@2x.png";

import smallScreenImage from "../../../assets/images/small-screen-160x80@2x.png";

interface ICoreLayoutProps {
  children: React.ReactNode;
  router: InjectedRouter; // v3
}

const expirationMessage = (
  <>
    Your license for Fleet Premium is about to expire. If you’d like to renew or
    have questions about downgrading,{" "}
    <a
      href="https://fleetdm.com/docs/using-fleet/faq#how-do-i-downgrade-from-fleet-premium-to-fleet-free"
      target="_blank"
      rel="noopener noreferrer"
    >
      please head to the Fleet documentation
      <img src={ExternalLinkIcon} alt="Open external link" id="new-tab-icon" />
    </a>
  </>
);

const CoreLayout = ({ children, router }: ICoreLayoutProps) => {
  const { config, currentUser, isPremiumTier } = useContext(AppContext);
  const { notification, hideFlash } = useContext(NotificationContext);
  const { setResetSelectedRows } = useContext(TableContext);
  const [showExpirationFlashMessage, setShowExpirationFlashMessage] = useState(
    false
  );

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

    setShowExpirationFlashMessage(
      licenseExpirationWarning(config?.license.expiration || "")
    );
  }, [notification]);

  const onLogoutUser = async () => {
    const { LOGOUT } = paths;
    router.push(LOGOUT);
  };

  const onNavItemClick = (path: string) => {
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
  const expirationNotification: INotification = {
    alertType: "warning-filled",
    isVisible: true,
    message: expirationMessage,
  };

  if (!currentUser || !config) {
    return null;
  }

  const { pathname } = global.window.location;

  return (
    <div className="app-wrap">
      <div className="overlay">
        <img src={smallScreenImage} alt="Unsupported screen size" />
        <div className="overlay__text">
          <h1>This screen size is not supported yet.</h1>
          <p>Please enlarge your browser or try again on a computer.</p>
        </div>
      </div>
      <nav className="site-nav">
        <SiteTopNav
          config={config}
          onLogoutUser={onLogoutUser}
          onNavItemClick={onNavItemClick}
          pathname={pathname}
          currentUser={currentUser}
        />
      </nav>
      <div className="core-wrapper">
        {isPremiumTier && showExpirationFlashMessage && (
          <FlashMessage
            fullWidth={fullWidthFlash}
            notification={expirationNotification}
            onRemoveFlash={() =>
              setShowExpirationFlashMessage(!showExpirationFlashMessage)
            }
          />
        )}
        <FlashMessage
          fullWidth={fullWidthFlash}
          notification={notification}
          onRemoveFlash={hideFlash}
          onUndoActionClick={onUndoActionClick}
        />
        {children}
      </div>
    </div>
  );
};

export default CoreLayout;
