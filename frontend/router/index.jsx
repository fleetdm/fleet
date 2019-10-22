import React from 'react';
import { browserHistory, IndexRedirect, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import { syncHistoryWithStore } from 'react-router-redux';

import AdminAppSettingsPage from 'pages/admin/AppSettingsPage';
import AdminUserManagementPage from 'pages/admin/UserManagementPage';
import AllPacksPage from 'pages/packs/AllPacksPage';
import App from 'components/App';
import AuthenticatedAdminRoutes from 'components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from 'components/AuthenticatedRoutes';
import ConfirmInvitePage from 'pages/ConfirmInvitePage';
import ConfirmSSOInvitePage from 'pages/ConfirmSSOInvitePage';
import CoreLayout from 'layouts/CoreLayout';
import EditPackPage from 'pages/packs/EditPackPage';
import EmailTokenRedirect from 'components/EmailTokenRedirect';
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
import PATHS from 'router/paths';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path={PATHS.HOME} component={App}>
        <Route path="setup" component={RegistrationPage} />
        <Route path="login" component={LoginRoutes}>
          <Route path="invites/:invite_token" component={ConfirmInvitePage} />
          <Route path="ssoinvites/:invite_token" component={ConfirmSSOInvitePage} />
          <Route path="forgot" />
          <Route path="reset" />
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route path="email/change/:token" component={EmailTokenRedirect} />
          <Route path="logout" component={LogoutPage} />
          <Route component={CoreLayout}>
            <IndexRedirect to={PATHS.MANAGE_HOSTS} />
            <Route path="admin" component={AuthenticatedAdminRoutes}>
              <Route path="users" component={AdminUserManagementPage} />
              <Route path="settings" component={AdminAppSettingsPage} />
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
      <Route component={Kolide404} />
    </Router>
  </Provider>
);

export default routes;
