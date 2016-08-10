import React from 'react';
import { render } from 'react-dom';
import { Router, Route, IndexRoute, browserHistory } from 'react-router';
import { Promise } from 'when';
import App from '#app/components/app';

export function run() {
  window.Promise = window.Promise || Promise;
  window.self = window;
  require('whatwg-fetch');

  render((
  <Router history={browserHistory}>
    <Route path="/" component={App}></Route>
  </Router>
), document.getElementById('app'))

}

require('#css');
// Style live reloading
if (module.hot) {
  let c = 0;
  module.hot.accept('#css', () => {
    require('#css');
    const a = document.createElement('a');
    const link = document.querySelector('link[rel="stylesheet"]');
    a.href = link.href;
    a.search = '?' + c++;
    link.href = a.href;
  });
}
