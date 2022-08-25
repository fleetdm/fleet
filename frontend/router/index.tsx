import React from "react";
import {
  browserHistory,
  IndexRedirect,
  IndexRoute,
  Route,
  RouteComponent,
  Router,
} from "react-router";

import AdminAppSettingsPage from "pages/admin/AppSettingsPage";
import AdminIntegrationsPage from "pages/admin/IntegrationsPage";
import AdminUserManagementPage from "pages/admin/UserManagementPage";
import AdminTeamManagementPage from "pages/admin/TeamManagementPage";
import TeamDetailsWrapper from "pages/admin/TeamManagementPage/TeamDetailsWrapper";
import App from "components/App";
import ConfirmInvitePage from "pages/ConfirmInvitePage";
import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import CoreLayout from "layouts/CoreLayout";
import GatedLayout from "layouts/GatedLayout";
import DeviceUserPage from "pages/hosts/details/DeviceUserPage";
import EditPackPage from "pages/packs/EditPackPage";
import EmailTokenRedirect from "components/EmailTokenRedirect";
import ForgotPasswordPage from "pages/ForgotPasswordPage";
import HostDetailsPage from "pages/hosts/details/HostDetailsPage";
import Homepage from "pages/Homepage";
import LabelPage from "pages/LabelPage";
import LoginPage, { LoginPreviewPage } from "pages/LoginPage";
import LogoutPage from "pages/LogoutPage";
import ManageHostsPage from "pages/hosts/ManageHostsPage";
import ManageSoftwarePage from "pages/software/ManageSoftwarePage";
import ManageQueriesPage from "pages/queries/ManageQueriesPage";
import ManagePacksPage from "pages/packs/ManagePacksPage";
import ManagePoliciesPage from "pages/policies/ManagePoliciesPage";
import ManageSchedulePage from "pages/schedule/ManageSchedulePage";
import PackComposerPage from "pages/packs/PackComposerPage";
import PolicyPage from "pages/policies/PolicyPage";
import QueryPage from "pages/queries/QueryPage";
import RegistrationPage from "pages/RegistrationPage";
import ResetPasswordPage from "pages/ResetPasswordPage";
import SoftwareDetailsPage from "pages/software/SoftwareDetailsPage";
import ApiOnlyUser from "pages/ApiOnlyUser";
import Fleet403 from "pages/errors/Fleet403";
import Fleet404 from "pages/errors/Fleet404";
import UserSettingsPage from "pages/UserSettingsPage";
import SettingsWrapper from "pages/admin/SettingsWrapper/SettingsWrapper";
import MembersPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPage";
import AgentOptionsPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/AgentOptionsPage";
import PATHS from "router/paths";
import AppProvider from "context/app";
import RoutingProvider from "context/routing";

import AuthGlobalAdminRoutes from "./components/AuthGlobalAdminRoutes";
import AuthAnyAdminRoutes from "./components/AuthAnyAdminRoutes";
import AuthenticatedRoutes from "./components/AuthenticatedRoutes";
import UnauthenticatedRoutes from "./components/UnauthenticatedRoutes";
import AuthGlobalAdminMaintainerRoutes from "./components/AuthGlobalAdminMaintainerRoutes";
import AuthAnyMaintainerAnyAdminRoutes from "./components/AuthAnyMaintainerAnyAdminRoutes";
import PremiumRoutes from "./components/PremiumRoutes";

interface IAppWrapperProps {
  children: JSX.Element;
  location?: {
    pathname: string;
  };
}

// App.tsx needs the context for user and config
const AppWrapper = ({ children, location }: IAppWrapperProps) => (
  <AppProvider>
    <RoutingProvider>
      <App location={location}>{children}</App>
    </RoutingProvider>
  </AppProvider>
);

const routes = (
  <Router history={browserHistory}>
    <Route path={PATHS.ROOT} component={AppWrapper}>
      <Route component={UnauthenticatedRoutes as RouteComponent}>
        <Route component={GatedLayout}>
          <Route path="setup" component={RegistrationPage} />
          <Route path="previewlogin" component={LoginPreviewPage} />
          <Route path="login" component={LoginPage} />
          <Route
            path="login/invites/:invite_token"
            component={ConfirmInvitePage}
          />
          <Route
            path="login/ssoinvites/:invite_token"
            component={ConfirmSSOInvitePage}
          />
          <Route path="login/forgot" component={ForgotPasswordPage} />
          <Route path="login/reset" component={ResetPasswordPage} />
        </Route>
      </Route>
      <Route component={AuthenticatedRoutes as RouteComponent}>
        <Route path="email/change/:token" component={EmailTokenRedirect} />
        <Route path="logout" component={LogoutPage} />
        <Route component={CoreLayout}>
          <IndexRedirect to={"dashboard"} />
          <Route path="dashboard" component={Homepage} />
          <Route path="settings" component={AuthAnyAdminRoutes}>
            <IndexRedirect to={"/dashboard"} />
            <Route component={SettingsWrapper}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="organization" component={AdminAppSettingsPage} />
                <Route
                  path="organization/:section"
                  component={AdminAppSettingsPage}
                />
                <Route path="integrations" component={AdminIntegrationsPage} />
                <Route path="users" component={AdminUserManagementPage} />
                <Route component={PremiumRoutes}>
                  <Route path="teams" component={AdminTeamManagementPage} />
                </Route>
              </Route>
            </Route>
            <Route path="teams/:team_id" component={TeamDetailsWrapper}>
              <Route path="members" component={MembersPage} />
              <Route path="options" component={AgentOptionsPage} />
            </Route>
          </Route>
          <Route path="labels">
            <IndexRedirect to={"manage"} />
            <Route path=":label_id" component={LabelPage} />
            <Route path="new" component={LabelPage} />
          </Route>
          <Route path="hosts">
            <IndexRedirect to={"manage"} />
            <Route path="manage" component={ManageHostsPage} />
            <Route path="manage/labels/:label_id" component={ManageHostsPage} />
            <Route path="manage/:active_label" component={ManageHostsPage} />
            <Route
              path="manage/labels/:label_id/:active_label"
              component={ManageHostsPage}
            />
            <Route
              path="manage/:active_label/labels/:label_id"
              component={ManageHostsPage}
            />
            <Route path=":host_id" component={HostDetailsPage} />
          </Route>
          <Route path="software">
            <IndexRedirect to={"manage"} />
            <Route path="manage" component={ManageSoftwarePage} />
            <Route path=":software_id" component={SoftwareDetailsPage} />
          </Route>
          <Route component={AuthGlobalAdminMaintainerRoutes}>
            <Route path="packs">
              <IndexRedirect to={"manage"} />
              <Route path="manage" component={ManagePacksPage} />
              <Route path="new" component={PackComposerPage} />
              <Route path=":id">
                <IndexRoute component={EditPackPage} />
                <Route path="edit" component={EditPackPage} />
              </Route>
            </Route>
          </Route>
          <Route component={AuthAnyMaintainerAnyAdminRoutes}>
            <Route path="schedule">
              <IndexRedirect to={"manage"} />
              <Route path="manage" component={ManageSchedulePage} />
              <Route
                path="manage/teams/:team_id"
                component={ManageSchedulePage}
              />
            </Route>
          </Route>
          <Route path="queries">
            <IndexRedirect to={"manage"} />
            <Route path="manage" component={ManageQueriesPage} />
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="new" component={QueryPage} />
            </Route>
            <Route path=":id" component={QueryPage} />
          </Route>
          <Route path="policies">
            <IndexRedirect to={"manage"} />
            <Route path="manage" component={ManagePoliciesPage} />
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="new" component={PolicyPage} />
            </Route>
            <Route path=":id" component={PolicyPage} />
          </Route>
          <Route
            path="profile"
            component={UserSettingsPage as RouteComponent}
          />
        </Route>
      </Route>
      <Route path="/device/:device_auth_token" component={DeviceUserPage} />
    </Route>
    <Route path="/apionlyuser" component={ApiOnlyUser} />
    <Route path="/404" component={Fleet404} />
    <Route path="/403" component={Fleet403} />
    <Route path="*" component={Fleet404} />
  </Router>
);

export default routes;
