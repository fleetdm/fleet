import React from "react";
// import { IHostEndUser } from "interfaces/host";
// import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { IHost, IHostEndUser } from "interfaces/host";

// export const generateEmailValue = (endUsers: IHostEndUser[]) => {
//   return "test";
//   // const { idp_email, other_emails } = endUsers[;

//   // if (!idp_email && !other_emails.length) {
//   //   return DEFAULT_EMPTY_CELL_VALUE;
//   // }

//   // return idp_email;
// };

// export const generateChromeProfilesValue = (endUsers: IHostEndUser[]) => {
//   // const { idp_email, other_emails } = endUsers;
//   // if (!idp_email && !other_emails.length) {
//   //   return DEFAULT_EMPTY_CELL_VALUE;
//   // }
//   // return idp_email;
// };

// export const generateOtherEmailValue = (endUsers: IHostEndUser[]) => {
//   if (!idp_email && !other_emails.length) {
//     return DEFAULT_EMPTY_CELL_VALUE;
//   }

//   return other_emails.map((email) => email.email).join(", ");
// };

// eslint-disable-next-line import/prefer-default-export
export const generateFullNameTipContent = (endUsers: IHostEndUser[]) => {
  return (
    <>
      Connect your identity provider to Fleet on the{" "}
      <b>
        Settings {">"} Integrations {">"} IdP
      </b>{" "}
      page.
    </>
  );
};

export const generateGroupsTipContent = generateFullNameTipContent;
