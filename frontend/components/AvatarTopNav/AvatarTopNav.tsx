import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK_DARK } from "utilities/constants";

interface IAvatarUserInterface {
  gravatarURL?: string;
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
  const { gravatarURL } = user;

  const isDefaultAvatar = false;
  // TODO: Need to figure out how to check if the gravatarURL is the default
  // if (gravatarURL.indexOf("www.gravatar.com/avatar/") > -1) {
  //   isDefaultAvatar = false;
  // }

  console.log("DEFAULT_GRAVATAR_LINK_DARK: ", DEFAULT_GRAVATAR_LINK_DARK);

  return (
    <div className="avatar-wrapper">
      <img
        alt={"User avatar"}
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={DEFAULT_GRAVATAR_LINK_DARK}
        onError={onError}
        onLoad={onLoad}
      />
    </div>
  );
};

export default Avatar;
