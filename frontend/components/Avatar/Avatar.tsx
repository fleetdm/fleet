import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

interface IFleetAvatarProps {
  className?: string;
}

/**
 * a simple component that can be used to display a the Fleet logo as an avatar
 */
const FleetAvatar = ({ className }: IFleetAvatarProps) => {
  return (
    <svg
      data-testid="fleet-avatar"
      className={className}
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="16" cy="16" r="15.5" fill="white" stroke="#C5C7D1" />
      <path
        d="M10 12C11.1046 12 12 11.1046 12 10C12 8.89543 11.1046 8 10 8C8.89543 8 8 8.89543 8 10C8 11.1046 8.89543 12 10 12Z"
        fill="white"
      />
      <path
        d="M16 12C17.1046 12 18 11.1046 18 10C18 8.89543 17.1046 8 16 8C14.8954 8 14 8.89543 14 10C14 11.1046 14.8954 12 16 12Z"
        fill="white"
      />
      <path
        d="M22 12C23.1046 12 24 11.1046 24 10C24 8.89543 23.1046 8 22 8C20.8954 8 20 8.89543 20 10C20 11.1046 20.8954 12 22 12Z"
        fill="white"
      />
      <path
        d="M10 18C11.1046 18 12 17.1046 12 16C12 14.8954 11.1046 14 10 14C8.89543 14 8 14.8954 8 16C8 17.1046 8.89543 18 10 18Z"
        fill="white"
      />
      <path
        d="M16 18C17.1046 18 18 17.1046 18 16C18 14.8954 17.1046 14 16 14C14.8954 14 14 14.8954 14 16C14 17.1046 14.8954 18 16 18Z"
        fill="white"
      />
      <path
        d="M10 24C11.1046 24 12 23.1046 12 22C12 20.8954 11.1046 20 10 20C8.89543 20 8 20.8954 8 22C8 23.1046 8.89543 24 10 24Z"
        fill="white"
      />
      <path
        d="M10.75 12.5C11.7165 12.5 12.5 11.7165 12.5 10.75C12.5 9.7835 11.7165 9 10.75 9C9.7835 9 9 9.7835 9 10.75C9 11.7165 9.7835 12.5 10.75 12.5Z"
        fill="#63C740"
      />
      <path
        d="M16 12.5C16.9665 12.5 17.75 11.7165 17.75 10.75C17.75 9.7835 16.9665 9 16 9C15.0335 9 14.25 9.7835 14.25 10.75C14.25 11.7165 15.0335 12.5 16 12.5Z"
        fill="#5CABDF"
      />
      <path
        d="M21.25 12.5C22.2165 12.5 23 11.7165 23 10.75C23 9.7835 22.2165 9 21.25 9C20.2835 9 19.5 9.7835 19.5 10.75C19.5 11.7165 20.2835 12.5 21.25 12.5Z"
        fill="#D66C7B"
      />
      <path
        d="M10.75 17.75C11.7165 17.75 12.5 16.9665 12.5 16C12.5 15.0335 11.7165 14.25 10.75 14.25C9.7835 14.25 9 15.0335 9 16C9 16.9665 9.7835 17.75 10.75 17.75Z"
        fill="#C98DEF"
      />
      <path
        d="M16 17.75C16.9665 17.75 17.75 16.9665 17.75 16C17.75 15.0335 16.9665 14.25 16 14.25C15.0335 14.25 14.25 15.0335 14.25 16C14.25 16.9665 15.0335 17.75 16 17.75Z"
        fill="#FAA669"
      />
      <path
        d="M10.75 23C11.7165 23 12.5 22.2165 12.5 21.25C12.5 20.2835 11.7165 19.5 10.75 19.5C9.7835 19.5 9 20.2835 9 21.25C9 22.2165 9.7835 23 10.75 23Z"
        fill="#3AEFC3"
      />
    </svg>
  );
};

interface IAvatarUserInterface {
  gravatar_url?: string;
  gravatar_url_dark?: string;
}

interface IAvatarProps {
  className?: string;
  size?: string;
  user: IAvatarUserInterface;
  hasWhiteBackground?: boolean;
  /**
   * Set this to `true` to use the fleet avatar instead of the users gravatar.
   */
  useFleetAvatar?: boolean;
}

const baseClass = "avatar";

const Avatar = ({
  className,
  size,
  user,
  hasWhiteBackground,
  useFleetAvatar = false,
}: IAvatarProps) => {
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
      {useFleetAvatar ? (
        <FleetAvatar className={avatarClasses} />
      ) : (
        <img
          alt="User avatar"
          className={`${avatarClasses} ${
            isLoading || isError ? "default" : ""
          }`}
          src={gravatar_url || DEFAULT_GRAVATAR_LINK}
          onError={onError}
          onLoad={onLoad}
        />
      )}
    </div>
  );
};

export default Avatar;
