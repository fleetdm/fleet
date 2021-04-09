import endpoints from 'kolide/endpoints';
import helpers from 'kolide/helpers';
import Base from 'kolide/base';

export default (client) => {
  return {
    create: (formData) => {
      const { USERS } = endpoints;

      return client.authenticatedPost(client._endpoint(USERS), JSON.stringify(formData))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
    destroy: (user) => {
      const { USERS } = endpoints;
      const endpoint = `${client._endpoint(USERS)}/${user.id}`;

      return client.authenticatedDelete(endpoint);
    },
    forgotPassword: ({ email }) => {
      const { FORGOT_PASSWORD } = endpoints;
      const endpoint = client.baseURL + FORGOT_PASSWORD;

      return Base.post(endpoint, JSON.stringify({ email }));
    },
    changePassword: (passwordParams) => {
      const { CHANGE_PASSWORD } = endpoints;

      return client.authenticatedPost(client._endpoint(CHANGE_PASSWORD), JSON.stringify(passwordParams));
    },
    confirmEmailChange: (user, token) => {
      const { CONFIRM_EMAIL_CHANGE } = endpoints;
      const endpoint = client._endpoint(CONFIRM_EMAIL_CHANGE(token));

      return client.authenticatedGet(endpoint)
        .then((response) => {
          return { ...user, email: response.new_email };
        });
    },
    enable: (user, { enabled }) => {
      const { ENABLE_USER } = endpoints;
      const endpoint = client._endpoint(ENABLE_USER(user.id));

      return client.authenticatedPost(endpoint, JSON.stringify({ enabled }))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },

    // NOTE: this function signature is the same as entities/host#loadAll as this was quicker to just copy
    // over. Ideally we'd want to remove the `selected` argument when we have more time, but for now
    // is is left unused.
    loadAll: (page = 0, perPage = 100, selected = '', globalFilter = '', sortBy = []) => {
      const { USERS } = endpoints;

      // TODO: add this query param logic to client class
      const pagination = `page=${page}&per_page=${perPage}`;

      let orderKeyParam = '';
      let orderDirection = '';
      if (sortBy.length !== 0) {
        const sortItem = sortBy[0];
        orderKeyParam += `&order_key=${sortItem.id}`;
        orderDirection = sortItem.desc ? '&order_direction=desc' : '&order_direction=asc';
      }

      let searchQuery = '';
      if (globalFilter !== '') {
        searchQuery = `&query=${globalFilter}`;
      }

      const userEndpoint = `${USERS}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;
      return client.authenticatedGet(client._endpoint(userEndpoint))
        .then((response) => {
          const { users } = response;

          return users.map(u => helpers.addGravatarUrlToResource(u));
        });
    },
    me: () => {
      const { ME } = endpoints;
      const endpoint = client.baseURL + ME;

      return client.authenticatedGet(endpoint)
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
    performRequiredPasswordReset: ({ password }) => {
      // Perform a password reset for the currently logged in user that has had a reset required
      const { PERFORM_REQUIRED_PASSWORD_RESET } = endpoints;
      const endpoint = client.baseURL + PERFORM_REQUIRED_PASSWORD_RESET;

      return client.authenticatedPost(endpoint, JSON.stringify({ new_password: password }))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
    requirePasswordReset: (userId, { require }) => {
      const { USERS } = endpoints;
      const endpoint = client._endpoint(`${USERS}/${userId}/require_password_reset`);

      return client.authenticatedPost(endpoint, JSON.stringify({ require }))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
    resetPassword: (formData) => {
      const { RESET_PASSWORD } = endpoints;
      const endpoint = client.baseURL + RESET_PASSWORD;

      return Base.post(endpoint, JSON.stringify(formData));
    },
    update: (user, formData) => {
      const { USERS } = endpoints;
      const updateUserEndpoint = `${client.baseURL}${USERS}/${user.id}`;

      return client.authenticatedPatch(updateUserEndpoint, JSON.stringify(formData))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
    updateAdmin: (user, { admin }) => {
      const { UPDATE_USER_ADMIN } = endpoints;
      const endpoint = client._endpoint(UPDATE_USER_ADMIN(user.id));

      return client.authenticatedPost(endpoint, JSON.stringify({ admin }))
        .then(response => helpers.addGravatarUrlToResource(response.user));
    },
  };
};
