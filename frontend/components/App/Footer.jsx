import React from 'react';
import radium from 'radium';
import componentStyles from './styles';
import footerLogo from '../../../assets/images/footer-logo.svg';

const { footerStyles } = componentStyles;

const Footer = () => {
  return (
    <footer style={footerStyles}>
      <img alt="Kolide logo" src={footerLogo} />
    </footer>
  );
};

export default radium(Footer);
