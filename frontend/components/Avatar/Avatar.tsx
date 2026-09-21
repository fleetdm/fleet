import classnames from "classnames";
import React, { useState, useCallback } from "react";

import { COLORS } from "styles/var/colors";

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
      <circle cx="10" cy="10" r="2" fill="#63C740" />
      <circle cx="16" cy="10" r="2" fill="#5CABDF" />
      <circle cx="22" cy="10" r="2" fill="#D66C7B" />
      <circle cx="10" cy="16" r="2" fill="#C98DEF" />
      <circle cx="16" cy="16" r="2" fill="#FAA669" />
      <circle cx="10" cy="22" r="2" fill="#3AEFC3" />
    </svg>
  );
};

interface IAPIOnlyAvatar {
  className?: string;
}

const APIOnlyAvatar = ({ className }: IAPIOnlyAvatar) => {
  return (
    <svg
      data-testid="apionly-avatar"
      className={className}
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle
        cx="16"
        cy="16"
        r="15"
        fill="none"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="2"
      />
      <path
        d="M11.5 12.75L8 16L11.5 19.25"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M18 11L14.5 21"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M21 19.25L24.5 16L21 12.75"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

const DefaultAvatar = ({ className }: IAPIOnlyAvatar) => {
  return (
    <svg
      data-testid="default-avatar"
      className={className}
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle
        cx="16"
        cy="16"
        r="15.25"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="1.5"
      />
      <circle
        cx="16"
        cy="13"
        r="5.25"
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="1.5"
      />
      <path
        stroke={COLORS["ui-fleet-black-50"]}
        strokeWidth="1.5"
        d="M16 18.75c5.084 0 9.21 4.102 9.248 9.178-.01.017-.024.044-.052.08a2.945 2.945 0 0 1-.462.463c-.448.374-1.127.812-1.99 1.23-1.725.833-4.114 1.549-6.744 1.549s-5.019-.716-6.744-1.55c-.863-.417-1.542-.855-1.99-1.23a2.945 2.945 0 0 1-.462-.461c-.028-.037-.043-.064-.053-.081A9.25 9.25 0 0 1 16 18.75Z"
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
  // Set this to true to show the API Only avatar
  useApiOnlyAvatar?: boolean;
}

const baseClass = "avatar";

const Avatar = ({
  className,
  size,
  user,
  hasWhiteBackground,
  useApiOnlyAvatar = false,
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
    "has-white-background": !!hasWhiteBackground && !useApiOnlyAvatar,
  });
  const { gravatar_url } = user;

  let avatar;

  if (useFleetAvatar) {
    avatar = <FleetAvatar className={avatarClasses} />;
  } else if (useApiOnlyAvatar) {
    avatar = <APIOnlyAvatar className={avatarClasses} />;
  } else if (gravatar_url) {
    avatar = (
      <img
        alt="User avatar"
        className={`${avatarClasses} ${isLoading || isError ? "default" : ""}`}
        src={gravatar_url}
        onError={onError}
        onLoad={onLoad}
      />
    );
  } else {
    avatar = <DefaultAvatar className={avatarClasses} />;
  }

  return <div className="avatar-wrapper">{avatar}</div>;
};

export default Avatar;
