import React, { Component } from 'react';

import kolideLogo from '../../../assets/images/kolide-logo-condensed.svg';
import gopher from '../../../assets/images/404.svg';

const baseClass = 'kolide-404';

class Kolide404 extends Component {

  render () {
    return (
      <div className={baseClass}>
        <header className="primary-header">
          <a href="/">
            <img className="primary-header__logo" src={kolideLogo} alt="Kolide" />
          </a>
        </header>
        <main>
          <h1>I have no memory of this place...</h1>
          <h2>Error 404</h2>
          <p>You seem to have lost your way.</p>
          <p>Might we recommend going back on your browser or visiting the <a href="/">home page?</a></p>
          <div className="gopher-container">
            <img src={gopher} role="presentation" />
            <p>Need immediate assistance? <br />Contact <a href="mailto:support@kolide.co">support@kolide.co</a></p>
          </div>
        </main>
      </div>
    );
  }
}

export default Kolide404;
