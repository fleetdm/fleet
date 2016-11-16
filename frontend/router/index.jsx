import React from 'react';
import { browserHistory, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import { syncHistoryWithStore } from 'react-router-redux';

import AdminUserManagementPage from 'pages/Admin/UserManagementPage';
import App from 'components/App';
import AuthenticatedAdminRoutes from 'components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from 'components/AuthenticatedRoutes';
import CoreLayout from 'layouts/CoreLayout';
import ForgotPasswordPage from 'pages/ForgotPasswordPage';
import HomePage from 'pages/HomePage';
import LoginRoutes from 'components/LoginRoutes';
import LogoutPage from 'pages/LogoutPage';
import ManageHostsPage from 'pages/hosts/ManageHostsPage';
import NewHostPage from 'pages/hosts/NewHostPage';
import QueryPage from 'pages/queries/QueryPage';
import QueryPageWrapper from 'components/queries/QueryPageWrapper';
import RegistrationPage from 'pages/RegistrationPage';
import ResetPasswordPage from 'pages/ResetPasswordPage';
import store from 'redux/store';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={App}>
        <Route path="setup" component={RegistrationPage} />
        <Route path="login" component={LoginRoutes}>
          <Route path="forgot" component={ForgotPasswordPage} />
          <Route path="reset" component={ResetPasswordPage} />
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route path="logout" component={LogoutPage} />
          <Route component={CoreLayout}>
            <IndexRoute component={HomePage} />
            <Route path="admin" component={AuthenticatedAdminRoutes}>
              <Route path="users" component={AdminUserManagementPage} />
            </Route>
            <Route path="queries" component={QueryPageWrapper}>
              <Route path="new" component={QueryPage} />
              <Route path=":id" component={QueryPage} />
            </Route>
            <Route path="hosts">
              <Route path="new" component={NewHostPage} />
              <Route path="manage" component={ManageHostsPage} />
            </Route>
          </Route>
        </Route>
      </Route>
    </Router>
  </Provider>
);

export default routes;
