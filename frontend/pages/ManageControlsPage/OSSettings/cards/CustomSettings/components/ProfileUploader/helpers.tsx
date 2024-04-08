import React from "react";
import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

// TODO: mobileconfig parser is a work in progress and not yet used in production
// https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles#3234127
const parseMobileconfig = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsText(file);
    reader.onerror = (error) => {
      reject(error);
    };
    reader.onabort = (error) => {
      reject(error);
    };
    reader.onload = () => {
      try {
        // parse mobile as xml
        const xmlDoc = new DOMParser().parseFromString(
          reader.result as string,
          "text/xml"
        );
        // check for any parser errors
        const parserErrors = xmlDoc.getElementsByTagName("parsererror");
        if (parserErrors.length > 0) {
          console.warn("parserErrors", parserErrors);
          throw new Error("Invalid file: parser error");
        }
        // get the top-level object, we assume it is the first `<dict>` element in the `<plist>`
        // https://developer.apple.com/documentation/devicemanagement/toplevel
        const tlo = xmlDoc.getElementsByTagName("dict")?.[0];
        if (tlo?.parentElement?.tagName !== "plist") {
          throw new Error("Invalid file: missing plist");
        }
        // get the payload display name from the top-level object, note that there may be other
        // `<dict>` elements in the `<plist>`, some of which contain `<key>PayloadDisplayName</key>`
        // elements, but we ignore those for now
        const pdnKey = Array.from(tlo.children).find(
          (child) =>
            child.tagName === "key" &&
            child.textContent === "PayloadDisplayName"
        );
        const pdnVal =
          (pdnKey?.nextElementSibling?.tagName === "string" &&
            pdnKey?.nextElementSibling?.textContent) ||
          "";
        // if the payload display name is empty, use the file name
        const result = pdnVal || file.name;
        console.log("parseMobileconfig result: ", result);
        resolve(result);
      } catch (error) {
        console.error("error", error);
        reject(error);
      }
    };
  });
};

export const parseFile = async (file: File): Promise<[string, string]> => {
  // get the file name and extension
  const nameParts = file.name.split(".");
  const name = nameParts.slice(0, -1).join(".");
  const ext = nameParts.slice(-1)[0];

  switch (ext) {
    case "xml": {
      return [name, "Windows"];
    }
    case "mobileconfig": {
      // // TODO: enable this once mobileconfig parser is vetted
      // try {
      //   const parsedName = await parseMobileConfig(file);
      //   return [parsedName, "macOS"];
      // } catch (e) {
      //   console.log("error", e);
      //   return [name, "macOS"];
      // }
      return [name, "macOS"];
    }
    case "json": {
      return [name, "macOS"];
    }
    default: {
      throw new Error(`Invalid file type: ${ext}`);
    }
  }
};

export const listNamesFromSelectedLabels = (dict: Record<string, boolean>) => {
  return Object.entries(dict).reduce((acc, [labelName, isSelected]) => {
    if (isSelected) {
      acc.push(labelName);
    }
    return acc;
  }, [] as string[]);
};

export const DEFAULT_ERROR_MESSAGE =
  "Couldn’t add configuration profile. Please try again.";

/** We want to add some additional messageing to some of the error messages so
 * we add them in this function. Otherwise, we'll just return the error message from the
 * API.
 */
// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err?.data?.errors?.[0]?.reason;

  if (
    apiReason.includes(
      "The configuration profile can’t include BitLocker settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>Disk encryption</b>.
      </span>
    );
  }

  if (
    apiReason.includes(
      "The configuration profile can’t include Windows update settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>OS updates</b>.
      </span>
    );
  }
  return apiReason || DEFAULT_ERROR_MESSAGE;
};
