import React from 'react';
import radium from 'radium';
import Icon from '../icons/Icon';
import styles from '../../styles';

const { color, padding } = styles;

const Footer = () => {
  const style = {
    container: {
      alignItems: 'center',
      backgroundColor: color.darkGrey,
      display: 'flex',
      justifyContent: 'center',
      height: '74px',
    },
    textLogo: {
      height: '20px',
      marginLeft: padding.base,
      width: '104px',
    },
  };

  return (
    <div style={style.container}>
      <Icon name="kolideLogo" />
      <Icon name="kolideText" style={style.textLogo} variant="lightGrey" />
    </div>
  );
};

export default radium(Footer);
