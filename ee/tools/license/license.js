#!/usr/bin/env node

import fs from "fs";

import jwt from "jsonwebtoken";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

// Generate the JWT license key using the provided options.
const generate = (argv) => {
  const {
    privateKey: privateKeyPath,
    keyPassphrase: keyPassphrasePath,
    expiration,
    customer,
    devices,
    note,
    tier,
  } = argv;

  const unixExpiration = Math.floor(expiration / 1000);

  const privateKey = fs.readFileSync(privateKeyPath);
  const keyPassphrase = fs.readFileSync(keyPassphrasePath).toString().trim();

  const licensePayload = {
    iss: "Fleet Device Management Inc.",
    exp: unixExpiration,
    sub: customer,
    devices,
    note,
    tier,
  };
  console.log(
    jwt.sign(
      licensePayload,
      { key: privateKey, passphrase: keyPassphrase },
      { algorithm: "ES256" }
    )
  );
};

// Like Date.parse but error if date cannot be parsed.
const coerceDate = (str) => {
  const date = Date.parse(str);
  if (isNaN(date)) {
    throw new Error(`'${str}' cannot be parsed as date`);
  }
  return date;
};

yargs(hideBin(process.argv))
  .command(
    "generate",
    "Generate a new license key",
    (cmd) => {
      cmd.option({
        "private-key": {
          description: "Path to private key for signing",
          type: "string",
          required: true,
        },
        "key-passphrase": {
          description: "Path to file containing private key passphrase",
          type: "string",
          required: true,
        },
        expiration: {
          description: "Expiration timestamp of license",
          type: "string",
          coerce: coerceDate,
          required: true,
        },
        customer: {
          description: "Name of customer",
          type: "string",
          required: true,
        },
        devices: {
          description: "Licensed device count",
          type: "number",
          required: true,
        },
        note: {
          description: "Additional note to add to license",
          type: "string",
        },
        tier: {
          description: "License tier",
          default: "premium",
          type: "string",
        },
      });
    },
    generate
  )
  .demandCommand(1, "Command must be provided") // Require a command
  .strict() // Reject unknown flags
  .version(false) // Disable version option
  .help()
  .alias("help", "h")
  .parse();
