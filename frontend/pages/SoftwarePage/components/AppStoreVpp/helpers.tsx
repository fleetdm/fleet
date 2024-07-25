import React from "react";
import { getErrorReason } from "interfaces/errors";

const ADD_SOFTWARE_ERROR_PREFIX = "Couldn’t add software.";
const DEFAULT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;

const generateAlreadyAvailableMessage = (msg: string) => {
  const regex = new RegExp(
    `${ADD_SOFTWARE_ERROR_PREFIX} (.+) already.+on the (.+) team.`
  );

  const match = msg.match(regex);
  if (!match) return DEFAULT_ERROR_MESSAGE;

  return (
    <>
      {ADD_SOFTWARE_ERROR_PREFIX} <b>{match[1]}</b> already has software
      available for install on the <b>{match[2]}</b> team.{" "}
    </>
  );
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  const reason = getErrorReason(e);

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    return generateAlreadyAvailableMessage(reason);
  }
  return DEFAULT_ERROR_MESSAGE;
};
