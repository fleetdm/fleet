import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK_DARK } from "utilities/constants";

interface IAvatarUserInterface {
  gravatar_url_dark?: string;
}

export interface IAvatarInterface {
  className?: string;
  size?: string;
  user: IAvatarUserInterface;
}

const baseClass = "avatar";

const Avatar = ({ className, size, user }: IAvatarInterface): JSX.Element => {
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
  });
  const { gravatar_url_dark } = user;

  return (
    <div className="avatar-wrapper-top-nav">
      <img
        alt="User avatar"
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={gravatar_url_dark || DEFAULT_GRAVATAR_LINK_DARK}
        onError={onError}
        onLoad={onLoad}
      />
    </div>
  );
};

export default Avatar;
