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

class Avatar extends React.Component<IAvatarInterface, null> {
  render(): JSX.Element {
    const { className, size, user } = this.props;
    const isSmall = size !== undefined && size.toLowerCase() === "small";
    const avatarClasses = classnames(baseClass, className, {
      [`${baseClass}--${size}`]: isSmall,
    });
    const { gravatarURL } = user;

    return (
      <img alt="User Avatar" className={avatarClasses} src={gravatarURL} />
    );
  }
}

export default Avatar;
