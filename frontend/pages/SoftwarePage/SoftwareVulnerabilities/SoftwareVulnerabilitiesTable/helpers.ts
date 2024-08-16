import { IVulnerability } from "interfaces/vulnerability";

// Function to normalize a CVE entry by removing the "CVE-" prefix and converting to lowercase
export const normalizeCVE = (cve: string): string => {
  return cve.toLowerCase().replace(/^cve-/, "");
};

// Function to check if a normalized query matches any normalized CVE in the data
export const isCVEInData = (
  query?: string,
  vulnerabilities?: IVulnerability[] | null
): boolean => {
  if (query && vulnerabilities) {
    // Normalize the query
    const normalizedQuery = normalizeCVE(query);

    // Check if the normalized query matches any normalized CVE in the vulnerabilities list
    return vulnerabilities.some(
      (vulnerability) => normalizeCVE(vulnerability.cve) === normalizedQuery
    );
  }
  return false;
};

export const getExploitedVulnerabiltiesDropdownOptions = (
  isPremiumTier = false
) => {
  const disabledTooltipContent = "Available in Fleet Premium.";

  return [
    {
      disabled: false,
      label: "All vulnerabilities",
      value: false,
      helpText: "All vulnerabilities detected on your hosts.",
    },
    {
      disabled: !isPremiumTier,
      label: "Exploited vulnerabilities",
      value: true,
      helpText:
        "Vulnerabilities that have been actively exploited in the wild.",
      tooltipContent: !isPremiumTier && disabledTooltipContent,
    },
  ];
};
