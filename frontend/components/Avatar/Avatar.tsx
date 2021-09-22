import React, { useState, useCallback } from "react";
import classnames from "classnames";

interface IAvatarUserInterface {
  gravatarURL: string;
}

interface IAvatarInterface {
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

  const isSmall = size !== undefined && size.toLowerCase() === "small";
  const avatarClasses = classnames(baseClass, className, {
    [`${baseClass}--${size}`]: isSmall,
  });
  const { gravatarURL } = user;

  return (
    <div className={avatarClasses}>
      <img
        alt={!isLoading && !isError ? "User avatar" : ""}
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={gravatarURL}
        onError={onError}
        onLoad={onLoad}
      />
    </div>
  );
};

export default Avatar;
