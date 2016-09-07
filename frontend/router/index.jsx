import React from 'react';
import { Router, Route, browserHistory } from 'react-router';
import { syncHistoryWithStore } from 'react-router-redux';
import App from '../components/app';
import store from '../redux/store';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Router history={history}>
    <Route path="/" component={App} />
  </Router>
);

export default routes;
