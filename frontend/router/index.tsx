import React from "react";
import {
  browserHistory,
  IndexRedirect,
  IndexRoute,
  Route,
  RouteComponent,
  Router,
  Redirect,
} from "react-router";

import OrgSettingsPage from "pages/admin/OrgSettingsPage";
import AdminIntegrationsPage from "pages/admin/IntegrationsPage";
import AdminUserManagementPage from "pages/admin/UserManagementPage";
import AdminTeamManagementPage from "pages/admin/TeamManagementPage";
import TeamDetailsWrapper from "pages/admin/TeamManagementPage/TeamDetailsWrapper";
import App from "components/App";
import ConfirmInvitePage from "pages/ConfirmInvitePage";
import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import CoreLayout from "layouts/CoreLayout";
import DashboardPage from "pages/DashboardPage";
import DeviceUserPage from "pages/hosts/details/DeviceUserPage";
import EditPackPage from "pages/packs/EditPackPage";
import EmailTokenRedirect from "components/EmailTokenRedirect";
import ForgotPasswordPage from "pages/ForgotPasswordPage";
import GatedLayout from "layouts/GatedLayout";
import HostDetailsPage from "pages/hosts/details/HostDetailsPage";
import LabelPage from "pages/LabelPage";
import LoginPage, { LoginPreviewPage } from "pages/LoginPage";
import LogoutPage from "pages/LogoutPage";
import ManageHostsPage from "pages/hosts/ManageHostsPage";
import ManageQueriesPage from "pages/queries/ManageQueriesPage";
import ManagePacksPage from "pages/packs/ManagePacksPage";
import ManagePoliciesPage from "pages/policies/ManagePoliciesPage";
import NoAccessPage from "pages/NoAccessPage";
import PackComposerPage from "pages/packs/PackComposerPage";
import PolicyPage from "pages/policies/PolicyPage";
import QueryDetailsPage from "pages/queries/details/QueryDetailsPage";
import LiveQueryPage from "pages/queries/live/LiveQueryPage";
import EditQueryPage from "pages/queries/edit/EditQueryPage";
import RegistrationPage from "pages/RegistrationPage";
import ResetPasswordPage from "pages/ResetPasswordPage";
import MDMAppleSSOPage from "pages/MDMAppleSSOPage";
import MDMAppleSSOCallbackPage from "pages/MDMAppleSSOCallbackPage";
import ApiOnlyUser from "pages/ApiOnlyUser";
import Fleet403 from "pages/errors/Fleet403";
import Fleet404 from "pages/errors/Fleet404";
import AccountPage from "pages/AccountPage";
import SettingsWrapper from "pages/admin/AdminWrapper";
import ManageControlsPage from "pages/ManageControlsPage/ManageControlsPage";
import UsersPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/UsersPage/UsersPage";
import AgentOptionsPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/AgentOptionsPage";
import OSUpdates from "pages/ManageControlsPage/OSUpdates";
import OSSettings from "pages/ManageControlsPage/OSSettings";
import SetupExperience from "pages/ManageControlsPage/SetupExperience/SetupExperience";
import WindowsMdmPage from "pages/admin/IntegrationsPage/cards/MdmSettings/WindowsMdmPage";
import MacOSMdmPage from "pages/admin/IntegrationsPage/cards/MdmSettings/MacOSMdmPage";
import Scripts from "pages/ManageControlsPage/Scripts/Scripts";
import WindowsAutomaticEnrollmentPage from "pages/admin/IntegrationsPage/cards/AutomaticEnrollment/WindowsAutomaticEnrollmentPage";
import HostQueryReport from "pages/hosts/details/HostQueryReport";
import SoftwarePage from "pages/SoftwarePage";
import SoftwareTitles from "pages/SoftwarePage/SoftwareTitles";
import SoftwareOS from "pages/SoftwarePage/SoftwareOS";
import SoftwareVulnerabilities from "pages/SoftwarePage/SoftwareVulnerabilities";
import SoftwareTitleDetailsPage from "pages/SoftwarePage/SoftwareTitleDetailsPage";
import SoftwareVersionDetailsPage from "pages/SoftwarePage/SoftwareVersionDetailsPage";
import TeamSettings from "pages/admin/TeamManagementPage/TeamDetailsWrapper/TeamSettings";
import SoftwareOSDetailsPage from "pages/SoftwarePage/SoftwareOSDetailsPage";
import SoftwareVulnerabilityDetailsPage from "pages/SoftwarePage/SoftwareVulnerabilityDetailsPage";

import PATHS from "router/paths";

import AppProvider from "context/app";
import RoutingProvider from "context/routing";

import AuthGlobalAdminRoutes from "./components/AuthGlobalAdminRoutes";
import AuthAnyAdminRoutes from "./components/AuthAnyAdminRoutes";
import AuthenticatedRoutes from "./components/AuthenticatedRoutes";
import UnauthenticatedRoutes from "./components/UnauthenticatedRoutes";
import AuthGlobalAdminMaintainerRoutes from "./components/AuthGlobalAdminMaintainerRoutes";
import AuthAnyMaintainerAnyAdminRoutes from "./components/AuthAnyMaintainerAnyAdminRoutes";
import AuthAnyMaintainerAdminObserverPlusRoutes from "./components/AuthAnyMaintainerAdminObserverPlusRoutes";
import PremiumRoutes from "./components/PremiumRoutes";
import ExcludeInSandboxRoutes from "./components/ExcludeInSandboxRoutes";

interface IAppWrapperProps {
  children: JSX.Element;
  location?: any;
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
          <Route path="login/denied" component={NoAccessPage} />
          <Route path="mdm/sso/callback" component={MDMAppleSSOCallbackPage} />
          <Route path="mdm/sso" component={MDMAppleSSOPage} />
        </Route>
      </Route>
      <Route component={AuthenticatedRoutes as RouteComponent}>
        <Route path="email/change/:token" component={EmailTokenRedirect} />
        <Route path="logout" component={LogoutPage} />
        <Route component={CoreLayout}>
          <IndexRedirect to="/dashboard" />
          <Route path="dashboard" component={DashboardPage}>
            <Route path="linux" component={DashboardPage} />
            <Route path="mac" component={DashboardPage} />
            <Route path="windows" component={DashboardPage} />
            <Route path="chrome" component={DashboardPage} />
          </Route>
          <Route path="settings" component={AuthAnyAdminRoutes}>
            <IndexRedirect to="organization/info" />
            <Route component={SettingsWrapper}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="organization" component={OrgSettingsPage} />
                <Route
                  path="organization/:section"
                  component={OrgSettingsPage}
                />
                <Route path="integrations" component={AdminIntegrationsPage} />
                <Route
                  path="integrations/:section"
                  component={AdminIntegrationsPage}
                />
                <Route component={ExcludeInSandboxRoutes}>
                  <Route path="users" component={AdminUserManagementPage} />
                </Route>
                <Route component={PremiumRoutes}>
                  <Route path="teams" component={AdminTeamManagementPage} />
                </Route>
              </Route>
            </Route>
            <Route path="integrations/mdm/windows" component={WindowsMdmPage} />
            <Route path="integrations/mdm/apple" component={MacOSMdmPage} />
            <Route
              path="integrations/automatic-enrollment/windows"
              component={WindowsAutomaticEnrollmentPage}
            />
            <Route path="teams" component={TeamDetailsWrapper}>
              <Redirect from="members" to="users" />
              <Route path="users" component={UsersPage} />
              <Route path="options" component={AgentOptionsPage} />
              <Route path="settings" component={TeamSettings} />
            </Route>
            <Redirect from="teams/:team_id" to="teams" />
            <Redirect from="teams/:team_id/users" to="teams" />
            <Redirect from="teams/:team_id/options" to="teams" />
          </Route>
          <Route path="labels">
            <IndexRedirect to="new" />
            <Route path=":label_id" component={LabelPage} />
            <Route path="new" component={LabelPage} />
          </Route>
          <Route path="hosts">
            <IndexRedirect to="manage" />
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
            <Route path=":host_id" component={HostDetailsPage}>
              <Redirect from="schedule" to="queries" />
              <Route path="scripts" component={HostDetailsPage} />
              <Route path="software" component={HostDetailsPage} />
              <Route path="queries" component={HostDetailsPage} />
              <Route path=":query_id" component={HostQueryReport} />
              <Route path="policies" component={HostDetailsPage} />
            </Route>

            <Route
              // outside of '/hosts' nested routes to avoid react-tabs-specific routing issues
              path=":host_id/queries/:query_id"
              component={HostQueryReport}
            />
          </Route>
          <Route component={ExcludeInSandboxRoutes}>
            <Route path="controls" component={AuthAnyMaintainerAnyAdminRoutes}>
              <IndexRedirect to="os-updates" />
              <Route component={ManageControlsPage}>
                <Route path="os-updates" component={OSUpdates} />
                <Route path="os-settings" component={OSSettings} />
                <Route path="os-settings/:section" component={OSSettings} />
                <Route path="setup-experience" component={SetupExperience} />
                <Route path="scripts" component={Scripts} />
                <Route
                  path="setup-experience/:section"
                  component={SetupExperience}
                />
              </Route>
            </Route>
          </Route>
          <Route path="software">
            <IndexRedirect to="titles" />
            <Route component={SoftwarePage}>
              <Route path="titles" component={SoftwareTitles} />
              <Route path="versions" component={SoftwareTitles} />
              <Route path="os" component={SoftwareOS} />
              <Route
                path="vulnerabilities"
                component={SoftwareVulnerabilities}
              />
              {/* This redirect keeps the old software/:id working */}
              <Redirect from=":id" to="versions/:id" />
            </Route>
            <Route
              path="vulnerabilities/:cve"
              component={SoftwareVulnerabilityDetailsPage}
            />
            <Route path="titles/:id" component={SoftwareTitleDetailsPage} />
            <Route path="versions/:id" component={SoftwareVersionDetailsPage} />
            <Route path="os/:id" component={SoftwareOSDetailsPage} />
          </Route>
          <Route component={AuthGlobalAdminMaintainerRoutes}>
            <Route path="packs">
              <IndexRedirect to="manage" />
              <Route path="manage" component={ManagePacksPage} />
              <Route path="new" component={PackComposerPage} />
              <Route path=":id">
                <IndexRoute component={EditPackPage} />
                <Route path="edit" component={EditPackPage} />
              </Route>
            </Route>
          </Route>
          <Route path="queries">
            <IndexRedirect to="manage" />
            <Route path="manage" component={ManageQueriesPage} />
            <Route component={AuthAnyMaintainerAdminObserverPlusRoutes}>
              <Route path="new">
                <IndexRoute component={EditQueryPage} />
                <Route path="live" component={LiveQueryPage} />
              </Route>
            </Route>
            <Route path=":id">
              <IndexRoute component={QueryDetailsPage} />
              <Route path="edit" component={EditQueryPage} />
              <Route path="live" component={LiveQueryPage} />
            </Route>
          </Route>
          <Route path="policies">
            <IndexRedirect to="manage" />
            <Route path="manage" component={ManagePoliciesPage} />
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="new" component={PolicyPage} />
            </Route>
            <Route path=":id" component={PolicyPage} />
          </Route>
          <Redirect from="profile" to="account" /> {/* deprecated URL */}
          <Route path="account" component={AccountPage} />
        </Route>
      </Route>
      <Route path="device">
        <IndexRedirect to=":device_auth_token" />

        <Route component={DeviceUserPage}>
          <Route path=":device_auth_token" component={DeviceUserPage}>
            <Route path="software" component={DeviceUserPage} />
            <Route path="policies" component={DeviceUserPage} />
          </Route>
        </Route>
      </Route>
    </Route>
    <Route path="/apionlyuser" component={ApiOnlyUser} />
    <Route path="/404" component={Fleet404} />
    <Route path="/403" component={Fleet403} />
    <Route path="*" component={Fleet404} />
  </Router>
);

export default routes;
