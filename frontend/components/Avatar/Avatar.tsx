import * as React from 'react';
const classnames = require('classnames');

interface IAvatarUserInterface {
  gravatarURL: string;
}

interface IAvatarInterface {
  className?: string;
  size?: string;
  user: IAvatarUserInterface;
}

interface IAvatarState {}

const baseClass = 'avatar';

class Avatar extends React.Component<IAvatarInterface, IAvatarState> {
  render (): JSX.Element {
    const { className, size, user } = this.props;
    const isSmall = size && size.toLowerCase() === 'small';
    const avatarClasses = classnames(baseClass, className, {
      [`${baseClass}--${size}`]: isSmall,
    });
    const { gravatarURL } = user;

    return (
      <img
        alt='User Avatar'
        className={avatarClasses}
        src={gravatarURL}
      />
    );
  }
}

export default Avatar;
