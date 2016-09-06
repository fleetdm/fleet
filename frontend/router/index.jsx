import React from 'react';
import { render } from 'react-dom';
import { Router, Route, browserHistory } from 'react-router';
import { Promise } from 'when';
import App from '../components/app';

const window = global.window || {};
const document = global.document || {};

export function run() {
  window.Promise = window.Promise || Promise;
  window.self = window;

  const router = (
    <Router history={browserHistory}>
      <Route path="/" component={App} />
    </Router>
  );

  render(router, document.getElementById('app'));
}

// Style live reloading
if (module.hot) {
  let c = 0;
  module.hot.accept('#css', () => {
    const a = document.createElement('a');
    const link = document.querySelector('link[rel="stylesheet"]');
    a.href = link.href;
    a.search = `?${c++}`;
    link.href = a.href;
  });
}

export default { run };
