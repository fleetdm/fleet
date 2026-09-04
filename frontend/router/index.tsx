import React, { FC, Suspense } from "react";
import { noop } from "lodash";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";
import {
  browserHistory,
  IndexRedirect,
  IndexRoute,
  InjectedRouter,
  Route,
  RouteComponent,
  Router,
  Redirect,
} from "react-router";

import App from "components/App";
import Spinner from "components/Spinner";
import ConfirmInvitePage from "pages/ConfirmInvitePage";
import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
import MfaPage from "pages/MfaPage";
import CoreLayout from "layouts/CoreLayout";
import DeviceUserSSOErrorPage from "pages/DeviceUserSSOErrorPage";
import EmailTokenRedirect from "components/EmailTokenRedirect";
import ForgotPasswordPage from "pages/ForgotPasswordPage";
import GatedLayout from "layouts/GatedLayout";
import LoginPage, { LoginPreviewPage } from "pages/LoginPage";
import LogoutPage from "pages/LogoutPage";
import NoAccessPage from "pages/NoAccessPage";
import RegistrationPage from "pages/RegistrationPage";
import ResetPasswordPage from "pages/ResetPasswordPage";
import MDMAppleSSOPage from "pages/MDMAppleSSOPage";
import MDMAppleSSOCallbackPage from "pages/MDMAppleSSOCallbackPage";
import ApiOnlyUser from "pages/ApiOnlyUser";
import Fleet403 from "pages/errors/Fleet403";
import Fleet404 from "pages/errors/Fleet404";
import ErrorPageLayout from "layouts/ErrorPageLayout";
import AccountPage from "pages/AccountPage";

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

const CHUNK_RELOAD_KEY = "fleet:chunk-reload";

// sessionStorage throws in some locked-down contexts. A chunk failing to load
// should not be replaced by a storage error, so treat it as unavailable and
// simply do not retry.
const chunkReloadTried = () => {
  try {
    return Boolean(sessionStorage.getItem(CHUNK_RELOAD_KEY));
  } catch {
    return true;
  }
};

const setChunkReloadTried = (tried: boolean) => {
  try {
    if (tried) {
      sessionStorage.setItem(CHUNK_RELOAD_KEY, "1");
    } else {
      sessionStorage.removeItem(CHUNK_RELOAD_KEY);
    }
  } catch {
    // Nothing to do — see above.
  }
};

/**
 * A page whose code is fetched when the route is first entered, so it stays out
 * of the entry chunk.
 *
 * Suspense covers the download. Spinner has its own 250ms delay, so a chunk
 * that arrives quickly shows nothing at all and only a genuinely slow load
 * produces a spinner.
 *
 * A failed download is retried once by reloading, which is the remedy when a
 * deploy has replaced the chunk while this tab was open. The retry marker
 * survives that reload and is cleared only once some chunk loads successfully,
 * so a chunk that is genuinely gone fails a second time and propagates to the
 * error boundary in App rather than reloading again.
 */
const lazyPage = (load: () => Promise<{ default: RouteComponent }>) => {
  const Lazy = React.lazy(() =>
    load()
      .then((m) => {
        // Only a chunk that actually loaded clears the marker. Clearing it on
        // window load instead would reset the guard on the very reload it
        // triggered, turning a permanently missing chunk into a reload loop.
        setChunkReloadTried(false);
        return m;
      })
      .catch((err) => {
        if (chunkReloadTried()) {
          throw err;
        }
        setChunkReloadTried(true);
        window.location.reload();
        // Nothing should render while the reload is in flight.
        return new Promise<{ default: RouteComponent }>(noop);
      })
  );

  const LazyPage = (props: Record<string, unknown>) => (
    <Suspense fallback={<Spinner />}>
      <Lazy {...props} />
    </Suspense>
  );
  return LazyPage as RouteComponent;
};

const LazyEditQueryPage = lazyPage(
  () =>
    import(/* webpackChunkName: "queries" */ "pages/queries/edit/EditQueryPage")
);
const LazyLiveQueryPage = lazyPage(
  () =>
    import(/* webpackChunkName: "queries" */ "pages/queries/live/LiveQueryPage")
);
const LazyQueryDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "queries" */ "pages/queries/details/QueryDetailsPage"
    )
);
const LazyEditPolicyPage = lazyPage(
  () => import(/* webpackChunkName: "policies" */ "pages/policies/edit")
);
const LazyLivePolicyPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "policies" */ "pages/policies/live/LivePolicyPage"
    )
);
const LazyPolicyDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "policies" */ "pages/policies/details/PolicyDetailsPage"
    )
);
const LazyDashboardPage = lazyPage(
  () => import(/* webpackChunkName: "dashboard" */ "pages/DashboardPage")
);
const LazyOrgSettingsPage = lazyPage(
  () => import(/* webpackChunkName: "admin" */ "pages/admin/OrgSettingsPage")
);
const LazyAdminIntegrationsPage = lazyPage(
  () => import(/* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage")
);
const LazyManageHostsPage = lazyPage(
  () => import(/* webpackChunkName: "hosts" */ "pages/hosts/ManageHostsPage")
);
const LazyHostDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "hosts" */ "pages/hosts/details/HostDetailsPage"
    )
);
const LazyHostQueryReport = lazyPage(
  () =>
    import(
      /* webpackChunkName: "hosts" */ "pages/hosts/details/HostQueryReport"
    )
);
const LazyManageControlsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/ManageControlsPage"
    )
);
const LazyOSUpdates = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/OSUpdates"
    )
);
const LazyOSSettings = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/OSSettings"
    )
);
const LazySetupExperience = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/SetupExperience/SetupExperience"
    )
);
const LazyScripts = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/Scripts/Scripts"
    )
);
const LazyVariables = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/Variables/Variables"
    )
);
const LazyScriptBatchDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "controls" */ "pages/ManageControlsPage/Scripts/ScriptBatchDetailsPage"
    )
);
const LazySoftwarePage = lazyPage(
  () => import(/* webpackChunkName: "software" */ "pages/SoftwarePage")
);
const LazySoftwareInventory = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareInventory"
    )
);
const LazySoftwareOS = lazyPage(
  () =>
    import(/* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareOS")
);
const LazySoftwareVulnerabilities = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareVulnerabilities"
    )
);
const LazySoftwareLibrary = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareLibrary"
    )
);
const LazySelfServiceCategoriesPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareLibrary/SelfServiceCategoriesPage"
    )
);
const LazySoftwareTitleDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareTitleDetailsPage"
    )
);
const LazySoftwareVersionDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareVersionDetailsPage"
    )
);
const LazySoftwareOSDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareOSDetailsPage"
    )
);
const LazySoftwareVulnerabilityDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareVulnerabilityDetailsPage"
    )
);
const LazySoftwareAddPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareAddPage"
    )
);
const LazySoftwareFleetMaintained = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained"
    )
);
const LazySoftwareCustomPackage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareAddPage/SoftwareCustomPackage"
    )
);
const LazySoftwareAppStore = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareAddPage/SoftwareAppStore"
    )
);
const LazyFleetMaintainedAppDetailsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "software" */ "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage"
    )
);
const LazyManageQueriesPage = lazyPage(
  () =>
    import(/* webpackChunkName: "queries" */ "pages/queries/ManageQueriesPage")
);
const LazyManagePoliciesPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "policies" */ "pages/policies/ManagePoliciesPage"
    )
);
const LazyManageLabelsPage = lazyPage(
  () => import(/* webpackChunkName: "labels" */ "pages/labels/ManageLabelsPage")
);
const LazyNewLabelPage = lazyPage(
  () => import(/* webpackChunkName: "labels" */ "pages/labels/NewLabelPage")
);
const LazyEditLabelPage = lazyPage(
  () => import(/* webpackChunkName: "labels" */ "pages/labels/EditLabelPage")
);
const LazySettingsWrapper = lazyPage(
  () => import(/* webpackChunkName: "admin" */ "pages/admin/AdminWrapper")
);
const LazyAdminManageUsersPage = lazyPage(
  () => import(/* webpackChunkName: "admin" */ "pages/admin/ManageUsersPage")
);
const LazyCreateUserPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageUsersPage/CreateUserPage"
    )
);
const LazyCreateApiUserPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageUsersPage/CreateApiUserPage"
    )
);
const LazyEditUserPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageUsersPage/EditUserPage"
    )
);
const LazyAdminManageFleetsPage = lazyPage(
  () => import(/* webpackChunkName: "admin" */ "pages/admin/ManageFleetsPage")
);
const LazyTeamDetailsWrapper = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageFleetsPage/TeamDetailsWrapper"
    )
);
const LazyTeamSettings = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageFleetsPage/TeamDetailsWrapper/TeamSettings"
    )
);
const LazyUsersPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageFleetsPage/TeamDetailsWrapper/UsersPage/UsersPage"
    )
);
const LazyAgentOptionsPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/ManageFleetsPage/TeamDetailsWrapper/AgentOptionsPage"
    )
);
const LazyWindowsMdmPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/WindowsMdmPage"
    )
);
const LazyAppleMdmPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/AppleMdmPage"
    )
);
const LazyAndroidMdmPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/AndroidMdmPage"
    )
);
const LazyWindowsEnrollmentPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/WindowsAutomaticEnrollmentPage"
    )
);
const LazyMicrosoftGraphPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/MicrosoftGraphPage"
    )
);
const LazyAppleBusinessManagerPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/AppleBusinessManagerPage"
    )
);
const LazyVppPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "admin" */ "pages/admin/IntegrationsPage/cards/MdmSettings/VppPage"
    )
);
const LazyManagePacksPage = lazyPage(
  () => import(/* webpackChunkName: "packs" */ "pages/packs/ManagePacksPage")
);
const LazyPackComposerPage = lazyPage(
  () => import(/* webpackChunkName: "packs" */ "pages/packs/PackComposerPage")
);
const LazyEditPackPage = lazyPage(
  () => import(/* webpackChunkName: "packs" */ "pages/packs/EditPackPage")
);
const LazyDeviceUserPage = lazyPage(
  () =>
    import(
      /* webpackChunkName: "device-user" */ "pages/hosts/details/DeviceUserPage"
    )
);

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
  router: InjectedRouter;
}

const queryClient = new QueryClient();

// App.tsx needs the context for user and config. We also wrap the application
// component in the required query client provider for react-query. This
// will allow us to use react-query hooks in the application component.
const AppWrapper = ({ children, location, router }: IAppWrapperProps) => {
  return (
    <AppProvider>
      <RoutingProvider>
        <CustomQueryClientProvider client={queryClient}>
          <App location={location} router={router}>
            {children}
          </App>
        </CustomQueryClientProvider>
      </RoutingProvider>
    </AppProvider>
  );
};

const routes = (
  <Router history={browserHistory}>
    {/* Kept outside AppWrapper (and before the "/" route) so the App shell
    never tries to load a normal session for API-only users. */}
    <Route path="/apionlyuser" component={ApiOnlyUser} />
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
          <Route
            path="mdm/apple/account_driven_enroll/sso/:token"
            component={MDMAppleSSOPage}
          />
        </Route>
      </Route>
      <Route component={AuthenticatedRoutes as RouteComponent}>
        <Route path="email/change/:token" component={EmailTokenRedirect} />
        <Route path="logout" component={LogoutPage} />
        <Route component={CoreLayout}>
          <IndexRedirect to="dashboard" />
          <Route path="dashboard" component={LazyDashboardPage}>
            <Route path="linux" component={LazyDashboardPage} />
            <Route path="mac" component={LazyDashboardPage} />
            <Route path="windows" component={LazyDashboardPage} />
            <Route path="chrome" component={LazyDashboardPage} />
            <Route path="ios" component={LazyDashboardPage} />
            <Route path="ipados" component={LazyDashboardPage} />
            <Route path="android" component={LazyDashboardPage} />
          </Route>
          <Route path="settings" component={AuthAnyAdminRoutes}>
            <IndexRedirect to="organization/info" />
            <Route component={LazySettingsWrapper}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="organization" component={LazyOrgSettingsPage} />
                {/* Forward old routes to new */}
                <Redirect from="organization/sso" to="integrations/sso" />
                <Redirect
                  from="organization/host-status-webhook"
                  to="integrations/host-status-webhook"
                />
                <Route
                  path="organization/:section"
                  component={LazyOrgSettingsPage}
                />
                <Route
                  path="integrations"
                  component={LazyAdminIntegrationsPage}
                />
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
                  component={LazyAdminIntegrationsPage}
                />
                <Route
                  path="integrations/sso/:subsection"
                  component={LazyAdminIntegrationsPage}
                />
                <Route component={ExcludeInSandboxRoutes}>
                  <Route path="users" component={LazyAdminManageUsersPage} />
                </Route>
                <Route component={PremiumRoutes}>
                  <Redirect from="teams" to="fleets" />
                  <Route path="fleets" component={LazyAdminManageFleetsPage} />
                </Route>
              </Route>
            </Route>
            <Route
              path="integrations/mdm/windows"
              component={LazyWindowsMdmPage}
            />
            <Route path="integrations/mdm/apple" component={LazyAppleMdmPage} />
            <Route
              path="integrations/mdm/android"
              component={LazyAndroidMdmPage}
            />
            {/* This redirect is used to handle old apple automatic enrollments page */}
            <Redirect
              from="integrations/automatic-enrollment/apple"
              to="integrations/mdm/ab"
            />
            {/* Redirect old /abm URL to /ab */}
            <Redirect from="integrations/mdm/abm" to="integrations/mdm/ab" />
            <Route
              path="integrations/mdm/ab"
              component={LazyAppleBusinessManagerPage}
            />
            <Route
              path="integrations/automatic-enrollment/windows"
              component={LazyWindowsEnrollmentPage}
            />
            <Route
              path="integrations/mdm/microsoft-graph"
              component={LazyMicrosoftGraphPage}
            />
            {/* This redirect is used to handle old vpp setup page */}
            <Redirect from="integrations/vpp/setup" to="integrations/mdm/vpp" />
            <Route path="integrations/mdm/vpp" component={LazyVppPage} />
            <Route component={ExcludeInSandboxRoutes}>
              <Route component={AuthGlobalAdminRoutes}>
                <Route path="users/new/human" component={LazyCreateUserPage} />
                <Route path="users/new/api" component={LazyCreateApiUserPage} />
                <Route
                  path="users/:user_id/edit"
                  component={LazyEditUserPage}
                />
              </Route>
            </Route>

            <Redirect from="teams" to="fleets" />
            <Redirect from="teams/users" to="fleets/users" />
            <Redirect from="teams/options" to="fleets/options" />
            <Redirect from="teams/settings" to="fleets/settings" />
            <Route path="fleets" component={LazyTeamDetailsWrapper}>
              <Redirect from="members" to="users" />
              <Route path="users" component={LazyUsersPage} />
              <Route path="options" component={LazyAgentOptionsPage} />
              <Route path="settings" component={LazyTeamSettings} />
            </Route>
            <Redirect from="teams/:team_id" to="fleets" />
            <Redirect from="teams/:team_id/users" to="fleets" />
            <Redirect from="teams/:team_id/options" to="fleets" />
          </Route>
          <Route path="labels">
            <IndexRedirect to="manage" />
            <Route path="manage" component={LazyManageLabelsPage} />
            <Route path="new" component={LazyNewLabelPage}>
              {/* maintaining previous 2 sub-routes for backward-compatibility of URL routes. NewLabelPage now sets the corresponding label type */}
              <Route path="dynamic" component={LazyNewLabelPage} />
              <Route path="manual" component={LazyNewLabelPage} />
            </Route>
            <Route path=":label_id" component={LazyEditLabelPage} />
          </Route>
          <Route path="hosts">
            <IndexRedirect to="manage" />
            <Route path="manage" component={LazyManageHostsPage} />
            <Route
              path="manage/labels/:label_id"
              component={LazyManageHostsPage}
            />
            <Route path=":host_id" component={LazyHostDetailsPage}>
              <IndexRedirect to="details" />
              <Route path="details" component={LazyHostDetailsPage} />
              <Route path="scripts" component={LazyHostDetailsPage} />
              <Route path="controls" component={LazyHostDetailsPage} />
              <Route path="software" component={LazyHostDetailsPage}>
                <IndexRedirect to="inventory" />
                <Route path="inventory" component={LazyHostDetailsPage} />
                <Route path="library" component={LazyHostDetailsPage} />
              </Route>
              <Route path="reports" component={LazyHostDetailsPage} />
              <Route path="policies" component={LazyHostDetailsPage} />
            </Route>

            <Redirect
              from=":host_id/queries/:query_id"
              to=":host_id/reports/:query_id"
            />
            <Route
              // outside of '/hosts' nested routes to avoid react-tabs-specific routing issues
              path=":host_id/reports/:query_id"
              component={LazyHostQueryReport}
            />
          </Route>
          <Route component={ExcludeInSandboxRoutes}>
            <Route
              path="controls"
              component={AuthAnyMaintainerAdminTechnicianRoutes}
            >
              <IndexRedirect to="os-updates" />
              <Route component={LazyManageControlsPage}>
                <Route path="os-updates" component={LazyOSUpdates} />
                <Route path="os-settings" component={LazyOSSettings} />
                <Redirect
                  from="os-settings/custom-settings"
                  to="os-settings/configuration-profiles"
                />
                <Route path="os-settings/:section" component={LazyOSSettings} />
                <Route
                  path="os-settings/:section/:platform"
                  component={LazyOSSettings}
                />

                <Route
                  path="setup-experience"
                  component={LazySetupExperience}
                />
                <Redirect
                  from="setup-experience/end-user-auth"
                  to="setup-experience/users"
                />
                <Route
                  path="setup-experience/:section"
                  component={LazySetupExperience}
                />
                <Route
                  path="setup-experience/:section/:platform"
                  component={LazySetupExperience}
                />

                <Route path="scripts">
                  <IndexRedirect to="library" />
                  <Route path=":section" component={LazyScripts} />
                </Route>
                <Route path="variables" component={LazyVariables} />
                <Route path="variables/:section" component={LazyVariables} />
              </Route>
            </Route>
            <Route
              path="controls/scripts/progress/:batch_execution_id"
              component={LazyScriptBatchDetailsPage}
            />
          </Route>
          <Route path="software">
            <IndexRedirect to="inventory" />
            {/* Legacy route redirect */}
            <Redirect from="titles" to="inventory" />
            {/* Check the add route first so 'software/add' isn't caught by title/version detail routes */}
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="add" component={LazySoftwareAddPage}>
                <IndexRedirect to="fleet-maintained" />
                <Route
                  path="fleet-maintained"
                  component={LazySoftwareFleetMaintained}
                />
                <Route path="app-store" component={LazySoftwareAppStore} />
                <Route path="package" component={LazySoftwareCustomPackage} />
              </Route>
              <Route
                path="add/fleet-maintained/:id"
                component={LazyFleetMaintainedAppDetailsPage}
              />
            </Route>
            <Route component={LazySoftwarePage}>
              <Route path="inventory" component={LazySoftwareInventory} />
              <Route path="versions" component={LazySoftwareInventory} />
              <Route path="os" component={LazySoftwareOS} />
              <Route
                path="vulnerabilities"
                component={LazySoftwareVulnerabilities}
              />
              <Route path="library" component={LazySoftwareLibrary} />
              {/* Legacy redirect: keeps old /software/:id URLs working */}
              <Redirect from=":id" to="versions/:id" />
            </Route>
            <Route
              path="library/categories"
              component={LazySelfServiceCategoriesPage}
            />
            <Route path="titles/:id" component={LazySoftwareTitleDetailsPage} />
            <Route
              path="versions/:id"
              component={LazySoftwareVersionDetailsPage}
            />
            <Route path="os/:id" component={LazySoftwareOSDetailsPage} />
            <Route
              path="vulnerabilities/:cve"
              component={LazySoftwareVulnerabilityDetailsPage}
            />
          </Route>
          <Route component={AuthGlobalAdminMaintainerRoutes}>
            <Route path="packs">
              <IndexRedirect to="manage" />
              <Route path="manage" component={LazyManagePacksPage} />
              <Route path="new" component={LazyPackComposerPage} />
              <Route path=":id">
                <IndexRoute component={LazyEditPackPage} />
                <Route path="edit" component={LazyEditPackPage} />
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
            <Route path="manage" component={LazyManageQueriesPage} />
            <Route component={AuthAnyMaintainerAdminObserverPlusRoutes}>
              <Route path="new">
                <IndexRoute component={LazyEditQueryPage} />
                <Route path="live" component={LazyLiveQueryPage} />
              </Route>
            </Route>
            <Route path=":id">
              <IndexRoute component={LazyQueryDetailsPage} />
              <Route path="edit" component={LazyEditQueryPage} />
              <Route path="live" component={LazyLiveQueryPage} />
            </Route>
          </Route>
          <Route path="policies">
            <IndexRedirect to="manage" />
            <Route path="manage" component={LazyManagePoliciesPage} />
            <Route component={AuthAnyMaintainerAnyAdminRoutes}>
              <Route path="new">
                <IndexRoute component={LazyEditPolicyPage} />
                <Route path="live" component={LazyLivePolicyPage} />
              </Route>
            </Route>
            <Route path=":id">
              <IndexRoute component={LazyPolicyDetailsPage} />
              <Route path="edit" component={LazyEditPolicyPage} />
              <Route path="live" component={LazyLivePolicyPage} />
            </Route>
          </Route>
          <Redirect from="profile" to="account" /> {/* deprecated URL */}
          <Route path="account" component={AccountPage} />
        </Route>
      </Route>
      <Route path="device">
        <IndexRedirect to=":device_auth_token" />
        <Route path="sso-error" component={DeviceUserSSOErrorPage} />
        <Route component={LazyDeviceUserPage}>
          <Route path=":device_auth_token" component={LazyDeviceUserPage}>
            <Route path="self-service" component={LazyDeviceUserPage} />
            <Route path="controls" component={LazyDeviceUserPage} />
            <Route path="software" component={LazyDeviceUserPage} />
            <Route path="policies" component={LazyDeviceUserPage} />
          </Route>
        </Route>
      </Route>
      {/* Inside AppWrapper so these render through App and can read the
      authenticated user from AppContext. The catch-all must stay last. */}
      <Route component={ErrorPageLayout}>
        <Route path="404" component={Fleet404} />
        <Route path="403" component={Fleet403} />
        <Route path="*" component={Fleet404} />
      </Route>
    </Route>
  </Router>
);

export default routes;
