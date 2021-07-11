// import React from "react";
// import {
//   browserHistory,
//   IndexRedirect,
//   IndexRoute,
//   Route,
//   Router,
// } from "react-router";
// import { Provider } from "react-redux";
// import { syncHistoryWithStore } from "react-router-redux";

// import AdminAppSettingsPage from "pages/admin/AppSettingsPage";
// import AdminUserManagementPage from "pages/admin/UserManagementPage";
// import AdminTeamManagementPage from "pages/admin/TeamManagementPage";
// import TeamDetailsWrapper from "pages/admin/TeamManagementPage/TeamDetailsWrapper";
// import AllPacksPage from "pages/packs/AllPacksPage";
// import App from "components/App";
// import AuthenticatedAdminRoutes from "components/AuthenticatedAdminRoutes";
// import AuthenticatedRoutes from "components/AuthenticatedRoutes";
// import AuthGlobalAdminMaintainerRoutes from "components/AuthGlobalAdminMaintainerRoutes";
// import AuthAnyMaintainerGlobalAdminRoutes from "components/AuthAnyMaintainerGlobalAdminRoutes";
// import BasicTierRoutes from "components/BasicTierRoutes";
// import ConfirmInvitePage from "pages/ConfirmInvitePage";
// import ConfirmSSOInvitePage from "pages/ConfirmSSOInvitePage";
// import CoreLayout from "layouts/CoreLayout";
// import EditPackPage from "pages/packs/EditPackPage";
// import EmailTokenRedirect from "components/EmailTokenRedirect";
// import HostDetailsPage from "pages/hosts/HostDetailsPage";
// import LoginRoutes from "components/LoginRoutes";
// import LogoutPage from "pages/LogoutPage";
// import ManageHostsPage from "pages/hosts/ManageHostsPage";
// import ManageQueriesPage from "pages/queries/ManageQueriesPage";
// import PackPageWrapper from "components/packs/PackPageWrapper";
// import PackComposerPage from "pages/packs/PackComposerPage";
// import QueryPage from "pages/queries/QueryPage";
// import QueryPageWrapper from "components/queries/QueryPageWrapper";
// import RegistrationPage from "pages/RegistrationPage";
// import ApiOnlyUser from "pages/ApiOnlyUser";
// import Fleet403 from "pages/Fleet403";
// import Fleet404 from "pages/Fleet404";
// import Fleet500 from "pages/Fleet500";
// import UserSettingsPage from "pages/UserSettingsPage";
// import SettingsWrapper from "pages/admin/SettingsWrapper/SettingsWrapper";
// import MembersPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPagePage";
// import AgentOptionsPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/AgentOptionsPage";
// import PATHS from "router/paths";
// import store from "redux/store";

// const history = syncHistoryWithStore(browserHistory, store);

// const routes = (
//   <Provider store={store}>
//     <Router history={history}>
//       <Route path={PATHS.HOME} component={App}>
//         <Route path="setup" component={RegistrationPage} />
//         <Route path="login" component={LoginRoutes}>
//           <Route path="invites/:invite_token" component={ConfirmInvitePage} />
//           <Route
//             path="ssoinvites/:invite_token"
//             component={ConfirmSSOInvitePage}
//           />
//           <Route path="forgot" />
//           <Route path="reset" />
//         </Route>
//         <Route component={AuthenticatedRoutes}>
//           <Route path="email/change/:token" component={EmailTokenRedirect} />
//           <Route path="logout" component={LogoutPage} />
//           <Route component={CoreLayout}>
//             <IndexRedirect to={PATHS.MANAGE_HOSTS} />
//             <Route path="settings" component={AuthenticatedAdminRoutes}>
//               <Route component={SettingsWrapper}>
//                 <Route path="organization" component={AdminAppSettingsPage} />
//                 <Route path="users" component={AdminUserManagementPage} />
//                 <Route component={BasicTierRoutes}>
//                   <Route path="teams" component={AdminTeamManagementPage} />
//                 </Route>
//               </Route>
//               <Route path="teams/:team_id" component={TeamDetailsWrapper}>
//                 <Route path="members" component={MembersPage} />
//                 <Route path="options" component={AgentOptionsPage} />
//               </Route>
//             </Route>
//             <Route path="hosts">
//               <Route path="manage" component={ManageHostsPage} />
//               <Route
//                 path="manage/labels/:label_id"
//                 component={ManageHostsPage}
//               />
//               <Route path="manage/:active_label" component={ManageHostsPage} />
//               <Route path=":host_id" component={HostDetailsPage} />
//             </Route>
//             <Route component={AuthGlobalAdminMaintainerRoutes}>
//               <Route path="packs" component={PackPageWrapper}>
//                 <Route path="manage" component={AllPacksPage} />
//                 <Route path="new" component={PackComposerPage} />
//                 <Route path=":id">
//                   <IndexRoute component={EditPackPage} />
//                   <Route path="edit" component={EditPackPage} />
//                 </Route>
//               </Route>
//             </Route>
//             <Route path="queries" component={QueryPageWrapper}>
//               <Route path="manage" component={ManageQueriesPage} />
//               <Route component={AuthAnyMaintainerGlobalAdminRoutes}>
//                 <Route path="new" component={QueryPage} />
//               </Route>
//               <Route path=":id" component={QueryPage} />
//             </Route>
//             <Route path="profile" component={UserSettingsPage} />
//           </Route>
//         </Route>
//       </Route>
//       <Route path="/apionlyuser" component={ApiOnlyUser} />
//       <Route path="/500" component={Fleet500} />
//       <Route path="/404" component={Fleet404} />
//       <Route path="/403" component={Fleet403} />
//       <Route path="*" component={Fleet404} />
//     </Router>
//   </Provider>
// );

// export default routes;

import React from "react";
import { Route, Switch, Redirect } from "react-router";
import { Provider } from "react-redux";
import { ConnectedRouter } from "connected-react-router";
import configureStore, { history } from "redux/store";

import AdminAppSettingsPage from "pages/admin/AppSettingsPage";
import AdminUserManagementPage from "pages/admin/UserManagementPage";
import AdminTeamManagementPage from "pages/admin/TeamManagementPage";
import TeamDetailsWrapper from "pages/admin/TeamManagementPage/TeamDetailsWrapper";
import AllPacksPage from "pages/packs/AllPacksPage";
import App from "components/App";
import AuthenticatedAdminRoutes from "components/AuthenticatedAdminRoutes";
import AuthenticatedRoutes from "components/AuthenticatedRoutes";
import AuthGlobalAdminMaintainerRoutes from "components/AuthGlobalAdminMaintainerRoutes";
import AuthAnyMaintainerGlobalAdminRoutes from "components/AuthAnyMaintainerGlobalAdminRoutes";
import BasicTierRoutes from "components/BasicTierRoutes";
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
import ApiOnlyUser from "pages/ApiOnlyUser";
import Fleet403 from "pages/Fleet403";
import Fleet404 from "pages/Fleet404";
import Fleet500 from "pages/Fleet500";
import UserSettingsPage from "pages/UserSettingsPage";
import SettingsWrapper from "pages/admin/SettingsWrapper/SettingsWrapper";
import MembersPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPagePage";
import AgentOptionsPage from "pages/admin/TeamManagementPage/TeamDetailsWrapper/AgentOptionsPage";
import PATHS from "router/paths";

const store = configureStore();

// const routes = (
//   <Provider store={store}>
//     <ConnectedRouter history={history}>
//       <Route path={PATHS.HOME} component={App}>
//         <Route path="setup" component={RegistrationPage} />
//         <Route path="login" component={LoginRoutes}>
//           <Route path="invites/:invite_token" component={ConfirmInvitePage} />
//           <Route
//             path="ssoinvites/:invite_token"
//             component={ConfirmSSOInvitePage}
//           />
//           <Route path="forgot" />
//           <Route path="reset" />
//         </Route>
//         <Route component={AuthenticatedRoutes}>
//           <Route path="email/change/:token" component={EmailTokenRedirect} />
//           <Route path="logout" component={LogoutPage} />
//           <Route component={CoreLayout}>
//             <Route
//               exact
//               path="/"
//               component={() => <Redirect to={PATHS.MANAGE_HOSTS} />}
//             />
//             <Route path="settings" component={AuthenticatedAdminRoutes}>
//               <Route component={SettingsWrapper}>
//                 <Route path="organization" component={AdminAppSettingsPage} />
//                 <Route path="users" component={AdminUserManagementPage} />
//                 <Route component={BasicTierRoutes}>
//                   <Route path="teams" component={AdminTeamManagementPage} />
//                 </Route>
//               </Route>
//               <Route path="teams/:team_id" component={TeamDetailsWrapper}>
//                 <Route path="members" component={MembersPage} />
//                 <Route path="options" component={AgentOptionsPage} />
//               </Route>
//             </Route>
//             <Route path="hosts">
//               <Route path="manage" component={ManageHostsPage} />
//               <Route
//                 path="manage/labels/:label_id"
//                 component={ManageHostsPage}
//               />
//               <Route
//                 path="manage/:active_label"
//                 component={ManageHostsPage}
//               />
//               <Route path=":host_id" component={HostDetailsPage} />
//             </Route>
//             <Route component={AuthGlobalAdminMaintainerRoutes}>
//               <Route path="packs" component={PackPageWrapper}>
//                 <Route path="manage" component={AllPacksPage} />
//                 <Route path="new" component={PackComposerPage} />
//                 <Route path=":id" component={EditPackPage}>
//                   <Route path="edit" component={EditPackPage} />
//                 </Route>
//               </Route>
//             </Route>
//             <Route path="queries" component={QueryPageWrapper}>
//               <Route path="manage" component={ManageQueriesPage} />
//               <Route component={AuthAnyMaintainerGlobalAdminRoutes}>
//                 <Route path="new" component={QueryPage} />
//               </Route>
//               <Route path=":id" component={QueryPage} />
//             </Route>
//             <Route path="profile" component={UserSettingsPage} />
//           </Route>
//         </Route>
//       </Route>
//       <Switch>
//         <Route path="/apionlyuser" component={ApiOnlyUser} />
//         <Route path="/500" component={Fleet500} />
//         <Route path="/404" component={Fleet404} />
//         <Route path="/403" component={Fleet403} />
//         <Route path="*" component={Fleet404} />
//       </Switch>
//     </ConnectedRouter>
//   </Provider>
// );

// const routes = (
//   <Provider store={store}>
//     <ConnectedRouter history={history}>
//       <Switch>
//         <Route
//           path={PATHS.HOME}
//           render={({ match: { path } }) => (
//             <App>
//               <Route path={`${path}/setup`} component={RegistrationPage} />
//               <Route
//                 path={`${path}/login`}
//                 render={({ match: { path } }) => (
//                   <LoginRoutes>
//                     <Route path={`${path}/invites/:invite_token`} component={ConfirmInvitePage} />
//                     <Route
//                       path={`${path}/ssoinvites/:invite_token`}
//                       component={ConfirmSSOInvitePage}
//                     />
//                     <Route path={`${path}/forgot`} />
//                     <Route path={`${path}/reset`} />
//                   </LoginRoutes>
//                 )}
//               />
//               <Route
//                 path="/"
//                 render={({ match: { path } }) => (
//                   <AuthenticatedRoutes>
//                     <Route path={`${path}/email/change/:token`} component={EmailTokenRedirect} />
//                     <Route path={`${path}/logout`} component={LogoutPage} />
//                     <Route
//                       path="/"
//                       render={({ match: { path } }) => (
//                         <CoreLayout>
//                           <Route
//                             exact
//                             path="/"
//                             component={() => <Redirect to={PATHS.MANAGE_HOSTS} />}
//                           />
//                           <Route
//                             path={`${path}/settings`}
//                             render={({ match: { path } }) => (
//                               <AuthenticatedAdminRoutes>
//                                 <Route
//                                   path="/"
//                                   render={({ match: { path } }) => (
//                                     <SettingsWrapper>
//                                       <Route path={`${path}/organization`} component={AdminAppSettingsPage} />
//                                       <Route path={`${path}/users`} component={AdminUserManagementPage} />
//                                       <Route
//                                         path="/"
//                                         render={({ match: { path } }) => (
//                                           <BasicTierRoutes>
//                                             <Route path={`${path}/teams`} component={AdminTeamManagementPage} />
//                                           </BasicTierRoutes>
//                                         )}
//                                       />
//                                     </SettingsWrapper>
//                                   )}
//                                 />
//                                 <Route path={`${path}/teams/:team_id`} component={TeamDetailsWrapper}>
//                                   <Route path="members" component={MembersPage} />
//                                   <Route path="options" component={AgentOptionsPage} />
//                                 </Route>
//                               </AuthenticatedAdminRoutes>
//                             )}
//                           />
//                           <Route path={`${path}/hosts`}>
//                             <Route path="manage" component={ManageHostsPage} />
//                             <Route
//                               path="manage/labels/:label_id"
//                               component={ManageHostsPage}
//                             />
//                             <Route
//                               path="manage/:active_label"
//                               component={ManageHostsPage}
//                             />
//                             <Route path=":host_id" component={HostDetailsPage} />
//                           </Route>
//                           <Route component={AuthGlobalAdminMaintainerRoutes}>
//                             <Route path="packs" component={PackPageWrapper}>
//                               <Route path="manage" component={AllPacksPage} />
//                               <Route path="new" component={PackComposerPage} />
//                               <Route path=":id" component={EditPackPage}>
//                                 <Route path="edit" component={EditPackPage} />
//                               </Route>
//                             </Route>
//                           </Route>
//                           <Route path={`${path}/queries`} component={QueryPageWrapper}>
//                             <Route path="manage" component={ManageQueriesPage} />
//                             <Route component={AuthAnyMaintainerGlobalAdminRoutes}>
//                               <Route path="new" component={QueryPage} />
//                             </Route>
//                             <Route path=":id" component={QueryPage} />
//                           </Route>
//                           <Route path={`${path}/profile`} component={UserSettingsPage} />
//                         </CoreLayout>
//                       )}
//                     />
//                   </AuthenticatedRoutes>
//                 )}
//               />
//             </App>
//           )}
//         />
//         <Route path="/apionlyuser" component={ApiOnlyUser} />
//         <Route path="/500" component={Fleet500} />
//         <Route path="/404" component={Fleet404} />
//         <Route path="/403" component={Fleet403} />
//         <Route path="*" component={Fleet404} />
//       </Switch>
//     </ConnectedRouter>
//   </Provider>
// );

const routes = (
  <Provider store={store}>
    <ConnectedRouter history={history}>
      <Switch>
        <Route path="/apionlyuser" component={ApiOnlyUser} />
        <Route path="/500" component={Fleet500} />
        <Route path="/404" component={Fleet404} />
        <Route path="/403" component={Fleet403} />

        <Route path={PATHS.HOME}>
          <App>
            <Switch>
              <Route path="/setup" component={RegistrationPage} />
              <Route
                path="/login"
                render={(props) => (
                  <LoginRoutes {...props}>
                    <Switch>
                      <Route
                        path="/invites/:invite_token"
                        component={ConfirmInvitePage}
                      />
                      <Route
                        path="/ssoinvites/:invite_token"
                        component={ConfirmSSOInvitePage}
                      />
                      <Route path="/forgot" />
                      <Route path="/reset" />
                    </Switch>
                  </LoginRoutes>
                )}
              />

              <Route
                render={(props) => (
                  <AuthenticatedRoutes {...props}>
                    <Switch>
                      <Route
                        path="/email/change/:token"
                        component={EmailTokenRedirect}
                      />
                      <Route path="/logout" component={LogoutPage} />

                      <Route
                        render={(props) => (
                          <CoreLayout {...props}>
                            <Switch>
                              <Route
                                exact
                                path="/"
                                component={() => (
                                  <Redirect to={PATHS.MANAGE_HOSTS} />
                                )}
                              />

                              <Route
                                path="/settings"
                                render={(props) => (
                                  <AuthenticatedAdminRoutes {...props}>
                                    <Switch>
                                      <Route
                                        render={(props) => (
                                          <SettingsWrapper {...props}>
                                            <Switch>
                                              <Route
                                                path="/organization"
                                                component={AdminAppSettingsPage}
                                              />
                                              <Route
                                                path="/users"
                                                component={
                                                  AdminUserManagementPage
                                                }
                                              />
                                              <Route
                                                render={(props) => (
                                                  <BasicTierRoutes {...props}>
                                                    <Route
                                                      path="/teams"
                                                      component={
                                                        AdminTeamManagementPage
                                                      }
                                                    />
                                                  </BasicTierRoutes>
                                                )}
                                              />
                                            </Switch>
                                          </SettingsWrapper>
                                        )}
                                      />

                                      <Route
                                        path="/teams/:team_id"
                                        render={(props) => (
                                          <TeamDetailsWrapper {...props}>
                                            <Switch>
                                              <Route
                                                path="/members"
                                                component={MembersPage}
                                              />
                                              <Route
                                                path="/options"
                                                component={AgentOptionsPage}
                                              />
                                            </Switch>
                                          </TeamDetailsWrapper>
                                        )}
                                      />
                                    </Switch>
                                  </AuthenticatedAdminRoutes>
                                )}
                              />

                              <Route
                                path="/hosts"
                                render={() => (
                                  <Switch>
                                    <Route
                                      path="/manage"
                                      component={ManageHostsPage}
                                    />
                                    <Route
                                      path="/manage/labels/:label_id"
                                      component={ManageHostsPage}
                                    />
                                    <Route
                                      path="/manage/:active_label"
                                      component={ManageHostsPage}
                                    />
                                    <Route
                                      path="/:host_id"
                                      component={HostDetailsPage}
                                    />
                                  </Switch>
                                )}
                              />

                              <Route
                                render={(props) => (
                                  <AuthGlobalAdminMaintainerRoutes {...props}>
                                    <Switch>
                                      <Route
                                        path="/packs"
                                        render={(props) => (
                                          <PackPageWrapper {...props}>
                                            <Switch>
                                              <Route
                                                path="/manage"
                                                component={AllPacksPage}
                                              />
                                              <Route
                                                path="/new"
                                                component={PackComposerPage}
                                              />
                                              <Route
                                                path="/:id"
                                                component={EditPackPage}
                                              />
                                              <Route
                                                path="/:id/edit"
                                                component={EditPackPage}
                                              />
                                            </Switch>
                                          </PackPageWrapper>
                                        )}
                                      />
                                    </Switch>
                                  </AuthGlobalAdminMaintainerRoutes>
                                )}
                              />

                              <Route
                                path="/queries"
                                render={(props) => (
                                  <QueryPageWrapper {...props}>
                                    <Switch>
                                      <Route
                                        path="/manage"
                                        component={ManageQueriesPage}
                                      />
                                      <Route
                                        render={(props) => (
                                          <AuthAnyMaintainerGlobalAdminRoutes
                                            {...props}
                                          >
                                            <Route
                                              path="/new"
                                              component={QueryPage}
                                            />
                                          </AuthAnyMaintainerGlobalAdminRoutes>
                                        )}
                                      />
                                      <Route
                                        path="/:id"
                                        component={QueryPage}
                                      />
                                    </Switch>
                                  </QueryPageWrapper>
                                )}
                              />

                              <Route
                                path="/profile"
                                component={UserSettingsPage}
                              />
                            </Switch>
                          </CoreLayout>
                        )}
                      />
                    </Switch>
                  </AuthenticatedRoutes>
                )}
              />
            </Switch>
          </App>
        </Route>

        <Route component={Fleet404} />
      </Switch>
    </ConnectedRouter>
  </Provider>
);

export default routes;
