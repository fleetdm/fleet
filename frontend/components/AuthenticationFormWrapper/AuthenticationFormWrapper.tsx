import React from "react";

import classnames from "classnames";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import FleetIcon from "../../../assets/images/fleet-avatar-24x24@2x.png";

interface IAuthenticationFormWrapperProps {
  children: React.ReactNode;
  header?: string;
  headerCta?: React.ReactNode;
  /** Only used on the registration page */
  breadcrumbs?: React.ReactNode;
  className?: string;
}

const baseClass = "auth-form-wrapper";

const AuthenticationFormWrapper = ({
  children,
  header,
  headerCta,
  breadcrumbs,
  className,
}: IAuthenticationFormWrapperProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className="app-wrap">
      <nav className="site-nav-container">
        <div className="site-nav-content">
          <ul className="site-nav-left">
            <li className="site-nav-item dup-org-logo" key="dup-org-logo">
              <div className="site-nav-item__logo-wrapper">
                <div className="site-nav-item__logo">
                  <OrgLogoIcon className="logo" src={FleetIcon} />
                </div>
              </div>
            </li>
          </ul>
        </div>
      </nav>
      {breadcrumbs}
      <div className={classNames}>
        <Card className={`${baseClass}__card`} paddingSize="xxlarge">
          {(header || headerCta) && (
            <div className={`${baseClass}__header-container`}>
              {header && <CardHeader header={header} />}
              {headerCta && headerCta}
            </div>
          )}
          {children}
        </Card>
      </div>
    </div>
  );
};

export default AuthenticationFormWrapper;
