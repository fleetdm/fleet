import SockJS from "sockjs-client";

import local from "utilities/local";

export default (client) => {
  return {
    queries: {
      run: (campaignID) => {
        return new Promise((resolve) => {
          const socket = new SockJS(
            `${client.baseURL}/v1/fleet/results`,
            undefined,
            {}
          );

          socket.onopen = () => {
            socket.send(
              JSON.stringify({
                type: "auth",
                data: { token: local.getItem("auth_token") },
              })
            );
            socket.send(
              JSON.stringify({
                type: "select_campaign",
                data: { campaign_id: campaignID },
              })
            );
          };

          return resolve(socket);
        });
      },
    },
  };
};
