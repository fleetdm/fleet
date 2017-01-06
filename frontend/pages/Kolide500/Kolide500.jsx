import React, { Component } from 'react';

import kolideLogo from '../../../assets/images/kolide-logo-condensed.svg';
import gopher from '../../../assets/images/500.svg';

const baseClass = 'kolide-500';

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
          <h1>Uh oh!</h1>
          <h2>Error 500</h2>
          <p>Something went wrong on our end.</p>
          <p>We have alerted the engineers and they are working on a solution.</p>
          <div className="gopher-container">
            <img src={gopher} role="presentation" />
            <p>Need immediate assistance? Contact <a href="mailto:support@kolide.co">support@kolide.co</a></p>
          </div>
        </main>
      </div>
    );
  }
}

export default Kolide404;
