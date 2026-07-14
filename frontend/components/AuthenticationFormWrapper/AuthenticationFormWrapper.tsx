import React from "react";

import classnames from "classnames";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import LogoOnlyNav from "components/top_nav/LogoOnlyNav";

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
        <LogoOnlyNav />
      </nav>
      {breadcrumbs}
      <div className={classNames}>
        <Card
          className={`${baseClass}__card`}
          borderRadiusSize="xxlarge"
          paddingSize="xlarge"
        >
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
