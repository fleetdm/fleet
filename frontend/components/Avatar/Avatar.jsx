import React, { PropTypes } from 'react';
import classnames from 'classnames';

import userInterface from '../../interfaces/user';

const Avatar = ({ size, className, style, user }) => {
  const { gravatarURL } = user;
  const isSmall = size && size.toLowerCase() === 'small';
  const avatarClasses = classnames(
    'avatar',
    { 'avatar--small': isSmall },
    className
  );

  return (
    <img
      alt="User avatar"
      src={gravatarURL}
      className={avatarClasses}
      style={style}
    />
  );
};

Avatar.propTypes = {
  size: PropTypes.string,
  className: PropTypes.string,
  style: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  user: userInterface.isRequired,
};

export default Avatar;
