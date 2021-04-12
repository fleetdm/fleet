import React from "react";
import {
  browserHistory,
  IndexRedirect,
  IndexRoute,
  Route,
  Router,
} from "react-router";
import { Provider } from "react-redux";
import { syncHistoryWithStore } from "react-router-redux";

import AdminAppSettingsPage from "pages/admin/AppSettingsPage";
import AdminUserManagementPage from "pages/admin/UserManagementPage";
import AdminOsqueryOptionsPage from "pages/admin/OsqueryOptionsPage";
import AllPacksPage from "pages/packs/AllPacksPage";
import App from "components/App";
import AuthenticatedAdminRoutes from "components/AuthenticatedAdminRoutes";
import AuthenticatedRoutes from "components/AuthenticatedRoutes";
import ConfirmInvitePage from "pages/ConfirmInvitePage";
import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import CoreLayout from "layouts/CoreLayout";
import EditPackPage from "pages/packs/EditPackPage";
import EmailTokenRedirect from "components/EmailTokenRedirect";
import HostDetailsPage from "pages/hosts/HostDetailsPage";
import LoginRoutes from "components/LoginRoutes";
import LogoutPage from "pages/LogoutPage";
import ManageHostsPage from "pages/hosts/ManageHostsPage";
import ManageQueriesPage from "pages/queries/ManageQueriesPage";
import PackPageWrapper from "components/packs/PackPageWrapper";
import PackComposerPage from "pages/packs/PackComposerPage";
import QueryPage from "pages/queries/QueryPage";
import QueryPageWrapper from "components/queries/QueryPageWrapper";
import RegistrationPage from "pages/RegistrationPage";
import Fleet404 from "pages/Fleet404";
import Fleet500 from "pages/Fleet500";
import UserSettingsPage from "pages/UserSettingsPage";
import SettingsWrapper from "pages/admin/SettingsWrapper/SettingsWrapper";
import PATHS from "router/paths";
import store from "redux/store";

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path={PATHS.HOME} component={App}>
        <Route path="setup" component={RegistrationPage} />
        <Route path="login" component={LoginRoutes}>
          <Route path="invites/:invite_token" component={ConfirmInvitePage} />
          <Route
            path="ssoinvites/:invite_token"
            component={ConfirmSSOInvitePage}
          />
          <Route path="forgot" />
          <Route path="reset" />
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route path="email/change/:token" component={EmailTokenRedirect} />
          <Route path="logout" component={LogoutPage} />
          <Route component={CoreLayout}>
            <IndexRedirect to={PATHS.MANAGE_HOSTS} />
            <Route path="settings" component={AuthenticatedAdminRoutes}>
              <Route component={SettingsWrapper}>
                <Route path="organization" component={AdminAppSettingsPage} />
                <Route path="users" component={AdminUserManagementPage} />
                <Route path="osquery" component={AdminOsqueryOptionsPage} />
              </Route>
            </Route>
            <Route path="hosts">
              <Route path="manage" component={ManageHostsPage} />
              <Route
                path="manage/labels/:label_id"
                component={ManageHostsPage}
              />
              <Route path="manage/:active_label" component={ManageHostsPage} />
              <Route path=":host_id" component={HostDetailsPage} />
            </Route>
            <Route path="packs" component={PackPageWrapper}>
              <Route path="manage" component={AllPacksPage} />
              <Route path="new" component={PackComposerPage} />
              <Route path=":id">
                <IndexRoute component={EditPackPage} />
                <Route path="edit" component={EditPackPage} />
              </Route>
            </Route>
            <Route path="queries" component={QueryPageWrapper}>
              <Route path="manage" component={ManageQueriesPage} />
              <Route path="new" component={QueryPage} />
              <Route path=":id" component={QueryPage} />
            </Route>
            <Route path="profile" component={UserSettingsPage} />
          </Route>
        </Route>
      </Route>
      <Route path="/500" component={Fleet500} />
      <Route path="/404" component={Fleet404} />
      <Route path="*" component={Fleet404} />
    </Router>
  </Provider>
);

export default routes;
