import React, { PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';
import userInterface from '../../interfaces/user';

const Avatar = ({ size, style, user }) => {
  const { gravatarURL } = user;

  return (
    <img
      alt="User avatar"
      src={gravatarURL}
      style={[componentStyles(size), style]}
    />
  );
};

Avatar.propTypes = {
  size: PropTypes.string,
  style: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  user: userInterface.isRequired,
};

export default radium(Avatar);
