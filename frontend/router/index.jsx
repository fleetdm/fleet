import React from 'react';
import { browserHistory, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import { syncHistoryWithStore } from 'react-router-redux';

import AdminAppSettingsPage from 'pages/Admin/AppSettingsPage';
import AdminUserManagementPage from 'pages/Admin/UserManagementPage';
import AllPacksPage from 'pages/packs/AllPacksPage';
import App from 'components/App';
import AuthenticatedAdminRoutes from 'components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from 'components/AuthenticatedRoutes';
import CoreLayout from 'layouts/CoreLayout';
import EditPackPage from 'pages/packs/EditPackPage';
import HomePage from 'pages/HomePage';
import LoginRoutes from 'components/LoginRoutes';
import LogoutPage from 'pages/LogoutPage';
import ManageHostsPage from 'pages/hosts/ManageHostsPage';
import NewHostPage from 'pages/hosts/NewHostPage';
import PackPageWrapper from 'components/packs/PackPageWrapper';
import PackComposerPage from 'pages/packs/PackComposerPage';
import QueryPage from 'pages/queries/QueryPage';
import QueryPageWrapper from 'components/queries/QueryPageWrapper';
import RegistrationPage from 'pages/RegistrationPage';
import store from 'redux/store';
import UserSettingsPage from 'pages/UserSettingsPage';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={App}>
        <Route path="setup" component={RegistrationPage} />
        <Route path="login" component={LoginRoutes}>
          <Route path="forgot" />
          <Route path="reset" />
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route path="logout" component={LogoutPage} />
          <Route component={CoreLayout}>
            <IndexRoute component={HomePage} />
            <Route path="admin" component={AuthenticatedAdminRoutes}>
              <Route path="users" component={AdminUserManagementPage} />
              <Route path="settings" component={AdminAppSettingsPage} />
            </Route>
            <Route path="hosts">
              <Route path="new" component={NewHostPage} />
              <Route path="manage" component={ManageHostsPage} />
            </Route>
            <Route path="packs" component={PackPageWrapper}>
              <Route path="all" component={AllPacksPage} />
              <Route path="new" component={PackComposerPage} />
              <Route path=":id">
                <IndexRoute component={EditPackPage} />
                <Route path="edit" component={EditPackPage} />
              </Route>
            </Route>
            <Route path="hosts">
              <Route path="new" component={NewHostPage} />
              <Route path="manage(/:active_label)" component={ManageHostsPage} />
            </Route>
            <Route path="queries" component={QueryPageWrapper}>
              <Route path="new" component={QueryPage} />
              <Route path=":id" component={QueryPage} />
            </Route>
            <Route path="settings" component={UserSettingsPage} />
          </Route>
        </Route>
      </Route>
    </Router>
  </Provider>
);

export default routes;
