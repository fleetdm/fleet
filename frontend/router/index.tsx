import React, { FC } from "react";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";
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
import CreateUserPage from "pages/admin/UserManagementPage/CreateUserPage";
import CreateApiUserPage from "pages/admin/UserManagementPage/CreateApiUserPage";
import EditUserPage from "pages/admin/UserManagementPage/EditUserPage";
import AdminTeamManagementPage from "pages/admin/TeamManagementPage";
import TeamDetailsWrapper from "pages/admin/TeamManagementPage/TeamDetailsWrapper";
import App from "components/App";
import ConfirmInvitePage from "pages/ConfirmInvitePage";
import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import MfaPage from "pages/MfaPage";
import CoreLayout from "layouts/CoreLayout";
import DashboardPage from "pages/DashboardPage";
import DeviceUserPage from "pages/hosts/details/DeviceUserPage";
import EditPackPage from "pages/packs/EditPackPage";
import EmailTokenRedirect from "components/EmailTokenRedirect";
import ForgotPasswordPage from "pages/ForgotPasswordPage";
import GatedLayout from "layouts/GatedLayout";
import HostDetailsPage from "pages/hosts/details/HostDetailsPage";
import ManageLabelsPage from "pages/labels/ManageLabelsPage";
import NewLabelPage from "pages/labels/NewLabelPage";
import EditLabelPage from "pages/labels/EditLabelPage";
import LoginPage, { LoginPreviewPage } from "pages/LoginPage";
import LogoutPage from "pages/LogoutPage";
import ManageHostsPage from "pages/hosts/ManageHostsPage";
import ManageQueriesPage from "pages/queries/ManageQueriesPage";
import ManagePacksPage from "pages/packs/ManagePacksPage";
import ManagePoliciesPage from "pages/policies/ManagePoliciesPage";
import NoAccessPage from "pages/NoAccessPage";
import PackComposerPage from "pages/packs/PackComposerPage";
import PolicyDetailsPage from "pages/policies/details/PolicyDetailsPage";
import EditPolicyPage from "pages/policies/edit";
import QueryDetailsPage from "pages/queries/details/QueryDetailsPage";
import LiveQueryPage from "pages/queries/live/LiveQueryPage";
import LivePolicyPage from "pages/policies/live/LivePolicyPage";
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
import AppleMdmPage from "pages/admin/IntegrationsPage/cards/MdmSettings/AppleMdmPage";
import AndroidMdmPage from "pages/admin/IntegrationsPage/cards/MdmSettings/AndroidMdmPage";
import Scripts from "pages/ManageControlsPage/Scripts/Scripts";
import Variables from "pages/ManageControlsPage/Variables/Variables";
import WindowsEnrollmentPage from "pages/admin/IntegrationsPage/cards/MdmSettings/WindowsAutomaticEnrollmentPage";
import AppleBusinessManagerPage from "pages/admin/IntegrationsPage/cards/MdmSettings/AppleBusinessManagerPage";
import VppPage from "pages/admin/IntegrationsPage/cards/MdmSettings/VppPage";
import HostQueryReport from "pages/hosts/details/HostQueryReport";
import SoftwarePage from "pages/SoftwarePage";
import SoftwareInventory from "pages/SoftwarePage/SoftwareInventory";
import SoftwareOS from "pages/SoftwarePage/SoftwareOS";
import SoftwareVulnerabilities from "pages/SoftwarePage/SoftwareVulnerabilities";
import SoftwareLibrary from "pages/SoftwarePage/SoftwareLibrary";
import SoftwareTitleDetailsPage from "pages/SoftwarePage/SoftwareTitleDetailsPage";
import SoftwareVersionDetailsPage from "pages/SoftwarePage/SoftwareVersionDetailsPage";
import TeamSettings from "pages/admin/TeamManagementPage/TeamDetailsWrapper/TeamSettings";
import SoftwareOSDetailsPage from "pages/SoftwarePage/SoftwareOSDetailsPage";
import SoftwareVulnerabilityDetailsPage from "pages/SoftwarePage/SoftwareVulnerabilityDetailsPage";
import SoftwareAddPage from "pages/SoftwarePage/SoftwareAddPage";
import SoftwareFleetMaintained from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained";
import SoftwareCustomPackage from "pages/SoftwarePage/SoftwareAddPage/SoftwareCustomPackage";
import SoftwareAppStore from "pages/SoftwarePage/SoftwareAddPage/SoftwareAppStore";
import FleetMaintainedAppDetailsPage from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage";
import ScriptBatchDetailsPage from "pages/ManageControlsPage/Scripts/ScriptBatchDetailsPage";

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
import AuthAnyMaintainerAdminTechnicianRoutes from "./components/AuthAnyMaintainerAdminTechnicianRoutes/AuthAnyMaintainerAdminTechnicianRoutes";
import PremiumRoutes from "./components/PremiumRoutes";
import ExcludeInSandboxRoutes from "./components/ExcludeInSandboxRoutes";

// We create a CustomQueryClientProvider that takes the same props as the original
// QueryClientProvider but adds the children prop as a ReactNode. This children
// prop is now required explicitly in React 18. We do it this way to avoid
// having to update the react-query package version and typings for now.
// When we upgrade React Query we should be able to remove this.
type ICustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: FC<ICustomQueryClientProviderProps> = QueryClientProvider;

interface IAppWrapperProps {
  children: JSX.Element;
  location?: any;
}

const queryClient = new QueryClient();

// App.tsx needs the context for user and config. We also wrap the application
// component in the required query client provider for react-query. This
// will allow us to use react-query hooks in the application component.
const AppWrapper = ({ children, location }: IAppWrapperProps) => {
  return (
    <AppProvider>
      <RoutingProvider>
        <CustomQueryClientProvider client={queryClient}>
          <App location={location}>{children}</App>
        </CustomQueryClientProvider>
      </RoutingProvider>
    </AppProvider>
  );
};

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
          <Route path="login/mfa/:token" component={MfaPage} />
          <Route path="login/forgot" component={ForgotPasswordPage} />
          <Route path="login/reset" component={ResetPasswordPage} />
          <Route path="login/denied" component={NoAccessPage} />
          <Route path="mdm/sso/callback" component={MDMAppleSSOCallbackPage} />
          <Route path="mdm/sso" component={MDMAppleSSOPage} />
          <Route
            path="mdm/apple/account_driven_enroll/sso"
            component={MDMAppleSSOPage}
          />
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
            <Route path="ios" component={DashboardPage} />
            <Route path="ipados" component={DashboardPage} />
            <Route path="android" component={DashboardPage} />
          </Route>
          <Route path="settings" component={AuthAnyAdminRoutes}>
            <IndexRedirect to="organization/info" />
            <Route component={SettingsWrapper}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="organization" component={OrgSettingsPage} />
                {/* Forward old routes to new */}
                <Redirect from="organization/sso" to="integrations/sso" />
                <Redirect
                  from="organization/host-status-webhook"
                  to="integrations/host-status-webhook"
                />
                <Route
                  path="organization/:section"
                  component={OrgSettingsPage}
                />
                <Route path="integrations" component={AdminIntegrationsPage} />
                {/* Forward old routes to new */}
                <Redirect
                  from="integrations/automatic-enrollment"
                  to="integrations/mdm"
                />
                <Redirect from="integrations/vpp" to="integrations/mdm" />
                <Redirect
                  from="integrations/sso"
                  to="integrations/sso/fleet-users"
                />
                <Route
                  path="integrations/:section"
                  component={AdminIntegrationsPage}
                />
                <Route
                  path="integrations/sso/:subsection"
                  component={AdminIntegrationsPage}
                />
                <Route component={ExcludeInSandboxRoutes}>
                  <Route path="users" component={AdminUserManagementPage} />
                </Route>
                <Route component={PremiumRoutes}>
                  <Redirect from="teams" to="fleets" />
                  <Route path="fleets" component={AdminTeamManagementPage} />
                </Route>
              </Route>
            </Route>
            <Route path="integrations/mdm/windows" component={WindowsMdmPage} />
            <Route path="integrations/mdm/apple" component={AppleMdmPage} />
            <Route path="integrations/mdm/android" component={AndroidMdmPage} />
            {/* This redirect is used to handle old apple automatic enrollments page */}
            <Redirect
              from="integrations/automatic-enrollment/apple"
              to="integrations/mdm/ab"
            />
            {/* Redirect old /abm URL to /ab */}
            <Redirect from="integrations/mdm/abm" to="integrations/mdm/ab" />
            <Route
              path="integrations/mdm/ab"
              component={AppleBusinessManagerPage}
            />
            <Route
              path="integrations/automatic-enrollment/windows"
              component={WindowsEnrollmentPage}
            />
            {/* This redirect is used to handle old vpp setup page */}
            <Redirect from="integrations/vpp/setup" to="integrations/mdm/vpp" />
            <Route path="integrations/mdm/vpp" component={VppPage} />
            <Route component={ExcludeInSandboxRoutes}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="users/new/human" component={CreateUserPage} />
                <Route path="users/new/api" component={CreateApiUserPage} />
                <Route path="users/:user_id/edit" component={EditUserPage} />
              </Route>
            </Route>

            <Redirect from="teams" to="fleets" />
            <Redirect from="teams/users" to="fleets/users" />
            <Redirect from="teams/options" to="fleets/options" />
            <Redirect from="teams/settings" to="fleets/settings" />
            <Route path="fleets" component={TeamDetailsWrapper}>
              <Redirect from="members" to="users" />
              <Route path="users" component={UsersPage} />
              <Route path="options" component={AgentOptionsPage} />
              <Route path="settings" component={TeamSettings} />
            </Route>
            <Redirect from="teams/:team_id" to="fleets" />
            <Redirect from="teams/:team_id/users" to="fleets" />
            <Redirect from="teams/:team_id/options" to="fleets" />
          </Route>
          <Route path="labels">
            <IndexRedirect to="manage" />
            <Route path="manage" component={ManageLabelsPage} />
            <Route path="new" component={NewLabelPage}>
              {/* maintaining previous 2 sub-routes for backward-compatibility of URL routes. NewLabelPage now sets the corresponding label type */}
              <Route path="dynamic" component={NewLabelPage} />
              <Route path="manual" component={NewLabelPage} />
            </Route>
            <Route path=":label_id" component={EditLabelPage} />
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
              <IndexRedirect to="details" />
              <Route path="details" component={HostDetailsPage} />
              <Route path="scripts" component={HostDetailsPage} />
              <Route path="software" component={HostDetailsPage}>
                <IndexRedirect to="inventory" />
                <Route path="inventory" component={HostDetailsPage} />
                <Route path="library" component={HostDetailsPage} />
              </Route>
              <Route path="reports" component={HostDetailsPage} />
              <Route path="policies" component={HostDetailsPage} />
            </Route>

            <Redirect
              from=":host_id/queries/:query_id"
              to=":host_id/reports/:query_id"
            />
            <Route
              // outside of '/hosts' nested routes to avoid react-tabs-specific routing issues
              path=":host_id/reports/:query_id"
              component={HostQueryReport}
            />
          </Route>
          <Route component={ExcludeInSandboxRoutes}>
            <Route
              path="controls"
              component={AuthAnyMaintainerAdminTechnicianRoutes}
            >
              <IndexRedirect to="os-updates" />
              <Route component={ManageControlsPage}>
                <Route path="os-updates" component={OSUpdates} />
                <Route path="os-settings" component={OSSettings} />
                <Redirect
                  from="os-settings/custom-settings"
                  to="os-settings/configuration-profiles"
                />
                <Route path="os-settings/:section" component={OSSettings} />

                <Route path="setup-experience" component={SetupExperience} />
                <Redirect
                  from="setup-experience/end-user-auth"
                  to="setup-experience/users"
                />
                <Route
                  path="setup-experience/:section"
                  component={SetupExperience}
                />
                <Route
                  path="setup-experience/:section/:platform"
                  component={SetupExperience}
                />

                <Route path="scripts">
                  <IndexRedirect to="library" />
                  <Route path=":section" component={Scripts} />
                </Route>
                <Route path="variables" component={Variables} />
              </Route>
            </Route>
            <Route
              path="controls/scripts/progress/:batch_execution_id"
              component={ScriptBatchDetailsPage}
            />
          </Route>
          <Route path="software">
            <IndexRedirect to="inventory" />
            {/* Legacy route redirect */}
            <Redirect from="titles" to="inventory" />
            {/* Check the add route first so 'software/add' isn't caught by title/version detail routes */}
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="add" component={SoftwareAddPage}>
                <IndexRedirect to="fleet-maintained" />
                <Route
                  path="fleet-maintained"
                  component={SoftwareFleetMaintained}
                />
                <Route path="app-store" component={SoftwareAppStore} />
                <Route path="package" component={SoftwareCustomPackage} />
              </Route>
              <Route
                path="add/fleet-maintained/:id"
                component={FleetMaintainedAppDetailsPage}
              />
            </Route>
            <Route component={SoftwarePage}>
              <Route path="inventory" component={SoftwareInventory} />
              <Route path="versions" component={SoftwareInventory} />
              <Route path="os" component={SoftwareOS} />
              <Route
                path="vulnerabilities"
                component={SoftwareVulnerabilities}
              />
              <Route path="library" component={SoftwareLibrary} />
              {/* Legacy redirect: keeps old /software/:id URLs working */}
              <Redirect from=":id" to="versions/:id" />
            </Route>
            <Route path="titles/:id" component={SoftwareTitleDetailsPage} />
            <Route path="versions/:id" component={SoftwareVersionDetailsPage} />
            <Route path="os/:id" component={SoftwareOSDetailsPage} />
            <Route
              path="vulnerabilities/:cve"
              component={SoftwareVulnerabilityDetailsPage}
            />
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
          <Redirect from="queries" to="reports" />
          <Redirect from="queries/manage" to="reports/manage" />
          <Redirect from="queries/new" to="reports/new" />
          <Redirect from="queries/new/live" to="reports/new/live" />
          <Redirect from="queries/:id" to="reports/:id" />
          <Redirect from="queries/:id/edit" to="reports/:id/edit" />
          <Redirect from="queries/:id/live" to="reports/:id/live" />
          <Route path="reports">
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
              <Route path="new">
                <IndexRoute component={EditPolicyPage} />
                <Route path="live" component={LivePolicyPage} />
              </Route>
            </Route>
            <Route path=":id">
              <IndexRoute component={PolicyDetailsPage} />
              <Route path="edit" component={EditPolicyPage} />
              <Route path="live" component={LivePolicyPage} />
            </Route>
          </Route>
          <Redirect from="profile" to="account" /> {/* deprecated URL */}
          <Route path="account" component={AccountPage} />
        </Route>
      </Route>
      <Route path="device">
        <IndexRedirect to=":device_auth_token" />
        <Route component={DeviceUserPage}>
          <Route path=":device_auth_token" component={DeviceUserPage}>
            <Route path="self-service" component={DeviceUserPage} />
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
