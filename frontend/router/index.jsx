import React from 'react';
import { browserHistory, IndexRedirect, IndexRoute, Redirect, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import { syncHistoryWithStore } from 'react-router-redux';

import AdminAppSettingsPage from 'pages/Admin/AppSettingsPage';
import AdminUserManagementPage from 'pages/Admin/UserManagementPage';
import AllPacksPage from 'pages/packs/AllPacksPage';
import App from 'components/App';
import AuthenticatedAdminRoutes from 'components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from 'components/AuthenticatedRoutes';
import ConfigOptionsPage from 'pages/config/ConfigOptionsPage';
import ConfirmInvitePage from 'pages/ConfirmInvitePage';
import CoreLayout from 'layouts/CoreLayout';
import EditPackPage from 'pages/packs/EditPackPage';
import LicensePage from 'pages/LicensePage';
import LoginRoutes from 'components/LoginRoutes';
import LogoutPage from 'pages/LogoutPage';
import ManageHostsPage from 'pages/hosts/ManageHostsPage';
import ManageQueriesPage from 'pages/queries/ManageQueriesPage';
import PackPageWrapper from 'components/packs/PackPageWrapper';
import PackComposerPage from 'pages/packs/PackComposerPage';
import QueryPage from 'pages/queries/QueryPage';
import QueryPageWrapper from 'components/queries/QueryPageWrapper';
import RegistrationPage from 'pages/RegistrationPage';
import Kolide404 from 'pages/Kolide404';
import Kolide500 from 'pages/Kolide500';
import store from 'redux/store';
import UserSettingsPage from 'pages/UserSettingsPage';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={App}>
        <Route path="setup" component={RegistrationPage} />
        <Route path="license" component={LicensePage} />
        <Route path="login" component={LoginRoutes}>
          <Route path="invites/:invite_token" component={ConfirmInvitePage} />
          <Route path="forgot" />
          <Route path="reset" />
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route path="logout" component={LogoutPage} />
          <Route component={CoreLayout}>
            <IndexRedirect to="/hosts/manage" />
            <Route path="admin" component={AuthenticatedAdminRoutes}>
              <Route path="users" component={AdminUserManagementPage} />
              <Route path="settings" component={AdminAppSettingsPage} />
            </Route>
            <Route path="config">
              <Route path="options" component={ConfigOptionsPage} />
            </Route>
            <Route path="hosts">
              <Route path="manage" component={ManageHostsPage} />
            </Route>
            <Route path="packs" component={PackPageWrapper}>
              <Route path="manage" component={AllPacksPage} />
              <Route path="new" component={PackComposerPage} />
              <Route path=":id">
                <IndexRoute component={EditPackPage} />
                <Route path="edit" component={EditPackPage} />
              </Route>
            </Route>
            <Route path="hosts">
              <Route path="manage(/:active_label)" component={ManageHostsPage} />
            </Route>
            <Route path="queries" component={QueryPageWrapper}>
              <Route path="manage" component={ManageQueriesPage} />
              <Route path="new" component={QueryPage} />
              <Route path=":id" component={QueryPage} />
            </Route>
            <Route path="settings" component={UserSettingsPage} />
          </Route>
        </Route>
      </Route>
      <Route path="/500" component={Kolide500} />
      <Route path="/404" component={Kolide404} />
      <Redirect from="*" to="/404" />
    </Router>
  </Provider>
);

export default routes;
