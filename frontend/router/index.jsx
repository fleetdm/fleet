import React from 'react';
import { browserHistory, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import radium, { StyleRoot } from 'radium';
import { syncHistoryWithStore } from 'react-router-redux';

import AdminUserManagementPage from '../pages/Admin/UserManagementPage';
import App from '../components/App';
import AuthenticatedAdminRoutes from '../components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from '../components/AuthenticatedRoutes';
import CoreLayout from '../layouts/CoreLayout';
import ForgotPasswordPage from '../pages/ForgotPasswordPage';
import HomePage from '../pages/HomePage';
import LoginRoutes from '../components/LoginRoutes';
import LogoutPage from '../pages/LogoutPage';
import ManageHostsPage from '../pages/hosts/ManageHostsPage';
import NewHostPage from '../pages/hosts/NewHostPage';
import NewQueryPage from '../pages/Queries/NewQueryPage';
import QueryPageWrapper from '../components/Queries/QueryPageWrapper';
import ResetPasswordPage from '../pages/ResetPasswordPage';
import store from '../redux/store';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <StyleRoot>
        <Route path="/" component={radium(App)}>
          <Route path="login" component={radium(LoginRoutes)}>
            <Route path="forgot" component={radium(ForgotPasswordPage)} />
            <Route path="reset" component={radium(ResetPasswordPage)} />
          </Route>
          <Route component={AuthenticatedRoutes}>
            <Route path="logout" component={radium(LogoutPage)} />
            <Route component={radium(CoreLayout)}>
              <IndexRoute component={radium(HomePage)} />
              <Route path="admin" component={AuthenticatedAdminRoutes}>
                <Route path="users" component={radium(AdminUserManagementPage)} />
              </Route>
              <Route path="queries" component={radium(QueryPageWrapper)}>
                <Route path="new" component={radium(NewQueryPage)} />
              </Route>
              <Route path="hosts">
                <Route path="new" component={radium(NewHostPage)} />
                <Route path="manage" component={radium(ManageHostsPage)} />
              </Route>
            </Route>
          </Route>
        </Route>
      </StyleRoot>
    </Router>
  </Provider>
);

export default routes;
