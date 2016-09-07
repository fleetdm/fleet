import React from 'react';
import { browserHistory, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import radium from 'radium';
import { syncHistoryWithStore } from 'react-router-redux';
import App from '../components/App';
import HomePage from '../pages/HomePage';
import store from '../redux/store';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={radium(App)}>
        <IndexRoute component={radium(HomePage)} />
      </Route>
    </Router>
  </Provider>
);

export default routes;
