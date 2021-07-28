import React from "react";
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
  const isSmall = size !== undefined && size.toLowerCase() === "small";
  const avatarClasses = classnames(baseClass, className, {
    [`${baseClass}--${size}`]: isSmall,
  });
  const { gravatarURL } = user;

  return <img alt="User Avatar" className={avatarClasses} src={gravatarURL} />;
};

export default Avatar;
