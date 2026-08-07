import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import paths from "router/paths";
import { QueryParams } from "utilities/url";

import SiteTopNav from "components/top_nav/SiteTopNav";
import LogoOnlyNav from "components/top_nav/LogoOnlyNav";

interface IErrorPageLayoutProps {
  children: React.ReactNode;
  router: InjectedRouter;
  location?: {
    pathname: string;
    search?: string;
    hash?: string;
    query: QueryParams;
  };
}

// Location is only used for nav active-link highlighting, which is irrelevant
// here, so an empty fallback is safe when no location is passed.
const FALLBACK_LOCATION = { pathname: "", query: {} };

const ErrorPageLayout = ({
  children,
  router,
  location,
}: IErrorPageLayoutProps) => {
  const { config, currentUser } = useContext(AppContext);

  const onLogoutUser = async () => {
    router.push(paths.LOGOUT);
  };

  const onUserMenuItemClick = (path: string) => {
    router.push(path);
  };

  const renderNav = () => {
    if (currentUser && config) {
      return (
        <SiteTopNav
          config={config}
          currentUser={currentUser}
          location={location ?? FALLBACK_LOCATION}
          onLogoutUser={onLogoutUser}
          onUserMenuItemClick={onUserMenuItemClick}
        />
      );
    }

    return <LogoOnlyNav to={paths.ROOT} />;
  };

  return (
    <div className="app-wrap">
      <nav className="site-nav-container">{renderNav()}</nav>
      <div className="error-page">{children}</div>
    </div>
  );
};

export default ErrorPageLayout;
