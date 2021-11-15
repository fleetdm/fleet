import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

interface IAvatarUserInterface {
  gravatarURL: string;
}

export interface IAvatarInterface {
  className?: string;
  size?: string;
  user: IAvatarUserInterface;
}

const baseClass = "avatar";

const Avatar = ({ className, size, user }: IAvatarInterface): JSX.Element => {
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isError, setIsError] = useState<boolean>();

  const onLoad = useCallback(() => {
    setIsLoading(false);
  }, []);
  const onError = useCallback(() => {
    setIsError(true);
  }, []);

  const avatarClasses = classnames(baseClass, className, {
    [`${baseClass}--${size?.toLowerCase()}`]: !!size,
  });
  const { gravatarURL } = user;

  return (
    <div>
      <img
        alt={!isLoading && !isError ? "User avatar" : ""}
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={gravatarURL || DEFAULT_GRAVATAR_LINK}
        onError={onError}
        onLoad={onLoad}
      />
    </div>
  );
};

export default Avatar;
