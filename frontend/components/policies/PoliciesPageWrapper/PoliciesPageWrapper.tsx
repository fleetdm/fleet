import React from "react";

const PoliciesPageWrapper = ({
  children,
}: {
  children: React.ReactNode;
}): React.ReactNode | null => {
  return children || null;
};

export default PoliciesPageWrapper;
