import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

interface IAvatarUserInterface {
  gravatar_url?: string;
  gravatar_url_dark?: string;
}

export interface IAvatarInterface {
  className?: string;
  size?: string;
  user: IAvatarUserInterface;
  hasWhiteBackground?: boolean;
}

const baseClass = "avatar";

const Avatar = ({
  className,
  size,
  user,
  hasWhiteBackground,
}: IAvatarInterface): JSX.Element => {
  const [isLoading, setIsLoading] = useState(true);
  const [isError, setIsError] = useState(false);

  const onLoad = useCallback(() => {
    setIsLoading(false);
  }, []);
  const onError = useCallback(() => {
    setIsError(true);
  }, []);

  const avatarClasses = classnames(baseClass, className, {
    [`${baseClass}--${size?.toLowerCase()}`]: !!size,
    "has-white-background": !!hasWhiteBackground,
  });
  const { gravatar_url } = user;

  return (
    <div className="avatar-wrapper">
      <img
        alt="User avatar"
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={gravatar_url || DEFAULT_GRAVATAR_LINK}
        onError={onError}
        onLoad={onLoad}
      />
    </div>
  );
};

export default Avatar;
