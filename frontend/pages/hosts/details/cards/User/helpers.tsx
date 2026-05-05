import React from "react";

import { IHostEndUser } from "interfaces/host";

export const generateUsernameValues = (endUsers: IHostEndUser[]) => {
  if (endUsers.length === 0) {
    return [];
  }

  return endUsers.reduce<string[]>((acc, endUser) => {
    if (endUser.idp_username) {
      acc.push(endUser.idp_username);
    }
    return acc;
  }, []);
};

export const generateFullNameValues = (endUsers: IHostEndUser[]) => {
  const endUser = endUsers[0];
  if (
    endUsers.length === 0 ||
    endUser.idp_info_updated_at === null ||
    endUser.idp_full_name === undefined
  ) {
    return [];
  }

  return [endUser.idp_full_name];
};

export const generateGroupsValues = (endUsers: IHostEndUser[]) => {
  const endUser = endUsers[0];
  if (
    endUsers.length === 0 ||
    endUser.idp_info_updated_at === null ||
    endUser.idp_groups === undefined
  ) {
    return [];
  }

  return endUser.idp_groups.sort((a, b) => {
    return a.localeCompare(b);
  });
};

export const generateChromeProfilesValues = (endUsers: IHostEndUser[]) => {
  const endUser = endUsers[0];
  if (endUsers.length === 0 || endUser.other_emails === undefined) {
    return [];
  }

  return endUser.other_emails.reduce<string[]>((acc, otherEmail) => {
    if (otherEmail.source === "google_chrome_profiles") {
      acc.push(otherEmail.email);
    }
    return acc;
  }, []);
};

export const generateOtherEmailsValues = (endUsers: IHostEndUser[]) => {
  const endUser = endUsers[0];
  if (endUsers.length === 0 || endUser.other_emails === undefined) {
    return [];
  }

  return endUser.other_emails.reduce<string[]>((acc, otherEmail) => {
    if (otherEmail.source === "custom") {
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
