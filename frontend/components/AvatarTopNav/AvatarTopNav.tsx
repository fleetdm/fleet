import React, { useState, useCallback } from "react";
import classnames from "classnames";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

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

  const isDefaultAvatar = true;
  // TODO: Need to figure out how to check if the gravatarURL is the default
  // if (gravatarURL.indexOf("www.gravatar.com/avatar/") > -1) {
  //   isDefaultAvatar = false;
  // }

  return (
    <div className="avatar-wrapper">
      {isDefaultAvatar ? (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <circle
            cx="12"
            cy="12"
            r="11.25"
            fill="#515774"
            stroke="white"
            strokeWidth="1.5"
          />
          <circle cx="12" cy="10.5" r="3.75" stroke="white" strokeWidth="1.5" />
          <path
            d="M18.7492 20.8922C18.7486 20.8929 18.748 20.8937 18.7475 20.8944C18.6939 20.9659 18.5929 21.0736 18.4304 21.2091C18.1081 21.4777 17.6139 21.798 16.9768 22.106C15.7041 22.7214 13.9402 23.25 12 23.25C10.0598 23.25 8.29593 22.7214 7.02318 22.106C6.38614 21.798 5.89185 21.4777 5.56965 21.2091C5.40709 21.0736 5.30606 20.9659 5.25252 20.8944C5.25195 20.8937 5.25139 20.8929 5.25084 20.8922C5.30844 17.214 8.30808 14.25 12 14.25C15.6919 14.25 18.6916 17.214 18.7492 20.8922ZM18.7802 20.8444C18.7804 20.8444 18.7792 20.8472 18.7758 20.853C18.7783 20.8474 18.7799 20.8445 18.7802 20.8444ZM5.21982 20.8444C5.22005 20.8445 5.22174 20.8474 5.22421 20.853C5.22083 20.8472 5.2196 20.8444 5.21982 20.8444Z"
            stroke="white"
            strokeWidth="1.5"
          />
        </svg>
      ) : (
        <img
          alt={"User avatar"}
          className={`${avatarClasses} ${
            isLoading || isError ? "default" : ""
          }`}
          src={gravatarURL}
          onError={onError}
          onLoad={onLoad}
        />
      )}
    </div>
  );
};

export default Avatar;
