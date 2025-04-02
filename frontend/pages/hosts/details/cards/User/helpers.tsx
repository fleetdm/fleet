import React from "react";

import { IHostEndUser } from "interfaces/host";

export const generateUsernameValues = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) {
    return [];
  }

  return endUsers.map((endUser) => {
    return endUser.idp_username;
  });
};

export const generateFullNameValues = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) {
    return [];
  }

  return endUsers.map((endUser) => {
    return endUser.idp_full_name;
  });
};

export const generateGroupsValues = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) {
    return [];
  }

  return endUsers[0].idp_groups.sort((a, b) => {
    return a.localeCompare(b);
  });
};

export const generateChromeProfilesValue = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) {
    return [];
  }

  return endUsers[0].other_emails.reduce<string[]>((acc, otherEmail) => {
    if (otherEmail.source === "google_chrome_profiles") {
      acc.push(otherEmail.email);
    }
    return acc;
  }, []);
};

export const generateFullNameTipContent = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) return null;

  if (endUsers[0].idp_info_updated_at === null) {
    return (
      <>
        Connect your identity provider to Fleet on the{" "}
        <b>
          Settings {">"} Integrations {">"} IdP
        </b>{" "}
        page.
      </>
    );
  }

  return <>This is the {'"givenName + familyName"'} from your IdP.</>;
};

export const generateGroupsTipContent = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) return null;

  if (endUsers[0].idp_info_updated_at === null) {
    return (
      <>
        Connect your identity provider to Fleet on the{" "}
        <b>
          Settings {">"} Integrations {">"} IdP
        </b>{" "}
        page.
      </>
    );
  }

  return null;
};
